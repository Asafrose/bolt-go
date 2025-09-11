package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// InstallProvider handles Slack OAuth installation flow
type InstallProvider struct {
	clientID                     string
	clientSecret                 string
	stateSecret                  string
	installationStore            InstallationStore
	authVersion                  string // v1 or v2
	logger                       *slog.Logger
	stateStore                   StateStore
	stateVerification            bool
	legacyStateVerification      bool
	stateCookieName              string
	stateCookieExpirationSeconds int
	directInstall                bool
	renderHtmlForInstallPath     func(*InstallURLOptions, *http.Request) string
	authorizationURL             string
}

// NewInstallProvider creates a new OAuth install provider
func NewInstallProvider(options InstallProviderOptions) (*InstallProvider, error) {
	if options.ClientID == "" {
		return nil, errors.New("clientID is required")
	}
	if options.ClientSecret == "" {
		return nil, errors.New("clientSecret is required")
	}

	provider := &InstallProvider{
		clientID:                     options.ClientID,
		clientSecret:                 options.ClientSecret,
		stateSecret:                  options.StateSecret,
		installationStore:            options.InstallationStore,
		authVersion:                  "v2", // default
		stateVerification:            true, // default
		stateCookieName:              "slack-app-oauth-state",
		stateCookieExpirationSeconds: 600, // 10 minutes
		authorizationURL:             "https://slack.com/oauth/v2/authorize",
	}

	// Set auth version
	if options.AuthVersion != "" {
		provider.authVersion = options.AuthVersion
		if options.AuthVersion == "v1" {
			provider.authorizationURL = "https://slack.com/oauth/authorize"
		}
	}

	// Set state verification
	if options.StateVerification != nil {
		provider.stateVerification = *options.StateVerification
	}

	// Set other options
	if options.LegacyStateVerification != nil {
		provider.legacyStateVerification = *options.LegacyStateVerification
	}
	if options.StateCookieName != "" {
		provider.stateCookieName = options.StateCookieName
	}
	if options.StateCookieExpirationSeconds > 0 {
		provider.stateCookieExpirationSeconds = options.StateCookieExpirationSeconds
	}
	if options.DirectInstall != nil {
		provider.directInstall = *options.DirectInstall
	}
	if options.RenderHtmlForInstallPath != nil {
		provider.renderHtmlForInstallPath = options.RenderHtmlForInstallPath
	}
	if options.AuthorizationURL != "" {
		provider.authorizationURL = options.AuthorizationURL
	}

	// Set logger
	if options.Logger != nil {
		if logger, ok := options.Logger.(*slog.Logger); ok {
			provider.logger = logger
		}
	}
	if provider.logger == nil {
		provider.logger = slog.Default()
	}

	// Set state store
	if options.StateStore != nil {
		provider.stateStore = options.StateStore
	} else if provider.stateVerification {
		if options.StateSecret != "" {
			provider.stateStore = NewEncryptedStateStore(options.StateSecret)
		} else {
			provider.stateStore = NewClearStateStore()
		}
	}

	// Set installation store
	if provider.installationStore == nil {
		provider.installationStore = NewMemoryInstallationStore()
	}

	return provider, nil
}

// GenerateInstallURL generates an OAuth installation URL
func (p *InstallProvider) GenerateInstallURL(ctx context.Context, options *InstallURLOptions, teamID string) (string, error) {
	if options == nil {
		options = &InstallURLOptions{}
	}

	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", p.clientID)

	// Set scopes
	if len(options.Scopes) > 0 {
		params.Set("scope", strings.Join(options.Scopes, ","))
	}
	if len(options.UserScopes) > 0 {
		params.Set("user_scope", strings.Join(options.UserScopes, ","))
	}

	// Set redirect URI
	if options.RedirectURI != "" {
		params.Set("redirect_uri", options.RedirectURI)
	}

	// Set team ID for direct install
	if teamID != "" {
		params.Set("team", teamID)
	}

	// Generate and set state parameter
	if p.stateVerification {
		state, err := p.stateStore.GenerateStateParam(ctx, options)
		if err != nil {
			return "", fmt.Errorf("failed to generate state parameter: %w", err)
		}
		params.Set("state", state)
	}

	return p.authorizationURL + "?" + params.Encode(), nil
}

// HandleInstallPath handles requests to the install path
func (p *InstallProvider) HandleInstallPath(req *http.Request, res http.ResponseWriter, installPathOptions *InstallPathOptions, installURLOptions *InstallURLOptions) error {
	ctx := req.Context()

	// Generate install URL
	installURL, err := p.GenerateInstallURL(ctx, installURLOptions, "")
	if err != nil {
		return fmt.Errorf("failed to generate install URL: %w", err)
	}

	// Handle direct install
	if p.directInstall {
		http.Redirect(res, req, installURL, http.StatusFound)
		return nil
	}

	// Render HTML page
	var html string
	if p.renderHtmlForInstallPath != nil {
		html = p.renderHtmlForInstallPath(installURLOptions, req)
	} else {
		html = p.defaultInstallPageHTML(installURL)
	}

	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)
	if _, err := res.Write([]byte(html)); err != nil {
		return err
	}

	return nil
}

// HandleCallback handles OAuth callback requests
func (p *InstallProvider) HandleCallback(req *http.Request, res http.ResponseWriter, callbackOptions *CallbackOptions, installURLOptions ...*InstallURLOptions) error {
	ctx := req.Context()

	// Parse query parameters
	query := req.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	errorParam := query.Get("error")

	// Check for OAuth errors
	if errorParam != "" {
		err := fmt.Errorf("OAuth error: %s", errorParam)
		if callbackOptions != nil && callbackOptions.Failure != nil {
			var options *InstallURLOptions
			if len(installURLOptions) > 0 {
				options = installURLOptions[0]
			}
			callbackOptions.Failure(err, options, req, res)
			return nil
		}
		return err
	}

	// Verify state parameter
	var verifiedOptions *InstallURLOptions
	if p.stateVerification && state != "" {
		var err error
		verifiedOptions, err = p.stateStore.VerifyStateParam(ctx, state)
		if err != nil {
			authErr := fmt.Errorf("state verification failed: %w", err)
			if callbackOptions != nil && callbackOptions.Failure != nil {
				callbackOptions.Failure(authErr, verifiedOptions, req, res)
				return nil
			}
			return authErr
		}
	} else if len(installURLOptions) > 0 {
		verifiedOptions = installURLOptions[0]
	}

	// Exchange code for token
	installation, err := p.exchangeCodeForToken(ctx, code, verifiedOptions)
	if err != nil {
		if callbackOptions != nil && callbackOptions.Failure != nil {
			callbackOptions.Failure(err, verifiedOptions, req, res)
			return nil
		}
		return err
	}

	// Store installation
	if err := p.installationStore.StoreInstallation(ctx, installation); err != nil {
		p.logger.Error("Failed to store installation", "error", err)
		storeErr := fmt.Errorf("failed to store installation: %w", err)
		if callbackOptions != nil && callbackOptions.Failure != nil {
			callbackOptions.Failure(storeErr, verifiedOptions, req, res)
			return nil
		}
		return storeErr
	}

	// Call success callback
	if callbackOptions != nil && callbackOptions.Success != nil {
		callbackOptions.Success(installation, verifiedOptions, req, res)
		return nil
	}

	// Default success response
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)
	if _, err := res.Write([]byte(p.defaultSuccessHTML())); err != nil {
		return err
	}

	return nil
}

// exchangeCodeForToken exchanges an authorization code for access tokens
func (p *InstallProvider) exchangeCodeForToken(ctx context.Context, code string, installOptions *InstallURLOptions) (*Installation, error) {
	var redirectURI string
	if installOptions != nil && installOptions.RedirectURI != "" {
		redirectURI = installOptions.RedirectURI
	}

	// Use slack SDK for OAuth token exchange
	httpClient := &http.Client{Timeout: 30 * time.Second}

	if p.authVersion == "v1" {
		// Use OAuth v1 API
		resp, err := slack.GetOAuthResponseContext(ctx, httpClient, p.clientID, p.clientSecret, code, redirectURI)
		if err != nil {
			return nil, fmt.Errorf("OAuth v1 token exchange failed: %w", err)
		}
		return p.convertOAuthV1Response(resp, installOptions), nil
	} else {
		// Use OAuth v2 API
		resp, err := slack.GetOAuthV2ResponseContext(ctx, httpClient, p.clientID, p.clientSecret, code, redirectURI)
		if err != nil {
			return nil, fmt.Errorf("OAuth v2 token exchange failed: %w", err)
		}
		return p.convertOAuthV2Response(resp, installOptions), nil
	}
}

// convertOAuthV2Response converts slack SDK OAuth v2 response to Installation
func (p *InstallProvider) convertOAuthV2Response(response *slack.OAuthV2Response, installOptions *InstallURLOptions) *Installation {
	// Build installation from slack SDK response
	installation := &Installation{
		IsEnterpriseInstall: response.IsEnterpriseInstall,
		AppID:               response.AppID,
		AuthVersion:         "v2",
		Scope:               response.Scope,
		AccessToken:         response.AccessToken,
		TokenType:           response.TokenType,
	}

	// Convert team information
	if response.Team.ID != "" {
		installation.Team = &Team{
			ID:   response.Team.ID,
			Name: response.Team.Name,
		}
	}

	// Convert enterprise information
	if response.Enterprise.ID != "" {
		installation.Enterprise = &Enterprise{
			ID:   response.Enterprise.ID,
			Name: response.Enterprise.Name,
		}
	}

	// Convert authed user information
	if response.AuthedUser.ID != "" {
		installation.AuthedUser = &AuthedUser{
			ID:          response.AuthedUser.ID,
			Scope:       response.AuthedUser.Scope,
			AccessToken: response.AuthedUser.AccessToken,
			TokenType:   response.AuthedUser.TokenType,
		}
	}

	// Convert incoming webhook information
	if response.IncomingWebhook.Channel != "" {
		installation.IncomingWebhook = &IncomingWebhook{
			Channel:          response.IncomingWebhook.Channel,
			ChannelID:        response.IncomingWebhook.ChannelID,
			ConfigurationURL: response.IncomingWebhook.ConfigurationURL,
			URL:              response.IncomingWebhook.URL,
		}
	}

	// Set bot information - OAuth v2 typically includes bot info in the main response
	if response.AccessToken != "" && response.BotUserID != "" {
		installation.Bot = &Bot{
			ID:          response.BotUserID,
			UserID:      response.BotUserID,
			AccessToken: response.AccessToken,
			TokenType:   response.TokenType,
			Scope:       response.Scope,
		}
		installation.BotToken = response.AccessToken
		installation.BotID = response.BotUserID
		installation.BotUserID = response.BotUserID
	}

	// Add metadata if provided
	if installOptions != nil && installOptions.Metadata != nil {
		installation.Metadata = installOptions.Metadata
	}

	return installation
}

// convertOAuthV1Response converts slack SDK OAuth v1 response to Installation
func (p *InstallProvider) convertOAuthV1Response(response *slack.OAuthResponse, installOptions *InstallURLOptions) *Installation {
	// Build installation from slack SDK response
	installation := &Installation{
		AuthVersion: "v1",
		AccessToken: response.AccessToken,
		Scope:       response.Scope,
	}

	// Convert team information
	if response.TeamID != "" {
		installation.Team = &Team{
			ID:   response.TeamID,
			Name: response.TeamName,
		}
	}

	// Convert bot information
	if response.Bot.BotAccessToken != "" {
		installation.Bot = &Bot{
			ID:          response.Bot.BotUserID,
			UserID:      response.Bot.BotUserID,
			AccessToken: response.Bot.BotAccessToken,
		}
		installation.BotToken = response.Bot.BotAccessToken
		installation.BotID = response.Bot.BotUserID
		installation.BotUserID = response.Bot.BotUserID
	}

	// Convert incoming webhook information
	if response.IncomingWebhook.URL != "" {
		installation.IncomingWebhook = &IncomingWebhook{
			Channel:          response.IncomingWebhook.Channel,
			ChannelID:        response.IncomingWebhook.ChannelID,
			ConfigurationURL: response.IncomingWebhook.ConfigurationURL,
			URL:              response.IncomingWebhook.URL,
		}
	}

	// Set user information
	if response.UserID != "" {
		installation.User = &User{
			ID:          response.UserID,
			TeamID:      response.TeamID,
			AccessToken: response.AccessToken,
			Scope:       response.Scope,
		}
	}

	// Add metadata if provided
	if installOptions != nil && installOptions.Metadata != nil {
		installation.Metadata = installOptions.Metadata
	}

	return installation
}

// defaultInstallPageHTML returns default HTML for the install page
func (p *InstallProvider) defaultInstallPageHTML(installURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Install Slack App</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; margin: 50px; }
        .install-button { 
            background-color: #4A154B; 
            color: white; 
            padding: 12px 24px; 
            text-decoration: none; 
            border-radius: 4px;
            display: inline-block;
            margin: 20px;
        }
        .install-button:hover { background-color: #611f69; }
    </style>
</head>
<body>
    <h1>Install Slack App</h1>
    <p>Click the button below to install this app to your Slack workspace.</p>
    <a href="%s" class="install-button">Add to Slack</a>
</body>
</html>`, installURL)
}

// defaultSuccessHTML returns default HTML for successful installation
func (p *InstallProvider) defaultSuccessHTML() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <title>Installation Successful</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; margin: 50px; }
        .success { color: #2eb886; }
    </style>
</head>
<body>
    <h1 class="success">✅ Installation Successful!</h1>
    <p>Your Slack app has been successfully installed.</p>
    <p>You can now close this window and return to Slack.</p>
</body>
</html>`
}
