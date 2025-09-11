package receivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/types"
)

// HTTPReceiver handles HTTP requests from Slack
type HTTPReceiver struct {
	signingSecret                 string
	endpoints                     *types.ReceiverEndpoints
	port                          int
	customRoutes                  []types.CustomRoute
	logger                        interface{}
	processBeforeResponse         bool
	signatureVerification         bool
	unhandledRequestHandler       http.HandlerFunc
	unhandledRequestTimeoutMillis int
	customProperties              map[string]interface{}

	// OAuth support
	installer              *oauth.InstallProvider
	installPath            string
	installRedirectURIPath string
	stateVerification      bool

	server *http.Server
	app    types.App
}

// NewHTTPReceiver creates a new HTTP receiver
func NewHTTPReceiver(options types.HTTPReceiverOptions) *HTTPReceiver {
	receiver := &HTTPReceiver{
		signingSecret:                 options.SigningSecret,
		endpoints:                     options.Endpoints,
		port:                          3000, // default port
		customRoutes:                  options.CustomRoutes,
		processBeforeResponse:         options.ProcessBeforeResponse,
		unhandledRequestTimeoutMillis: options.UnhandledRequestTimeoutMillis,
		signatureVerification:         true, // default to true
		customProperties:              options.CustomProperties,
		stateVerification:             true, // default to true
	}

	// Initialize OAuth if configuration is provided
	if options.ClientID != "" && options.ClientSecret != "" {
		// Create install provider options
		installProviderOptions := oauth.InstallProviderOptions{
			ClientID:     options.ClientID,
			ClientSecret: options.ClientSecret,
			StateSecret:  options.StateSecret,
		}

		// Set installation store if provided and valid
		if options.InstallationStore != nil {
			if store, ok := options.InstallationStore.(oauth.InstallationStore); ok {
				installProviderOptions.InstallationStore = store
			}
		}

		// Set installer options if provided
		if options.InstallerOptions != nil {
			installProviderOptions.StateVerification = options.InstallerOptions.StateVerification
			installProviderOptions.LegacyStateVerification = options.InstallerOptions.LegacyStateVerification
			installProviderOptions.StateCookieName = options.InstallerOptions.StateCookieName
			installProviderOptions.StateCookieExpirationSeconds = options.InstallerOptions.StateCookieExpirationSeconds
			installProviderOptions.AuthVersion = options.InstallerOptions.AuthVersion
			installProviderOptions.DirectInstall = options.InstallerOptions.DirectInstall
			installProviderOptions.AuthorizationURL = options.InstallerOptions.AuthorizationURL

			// Set paths
			receiver.installPath = options.InstallerOptions.InstallPath
			if receiver.installPath == "" {
				receiver.installPath = "/slack/install"
			}
			receiver.installRedirectURIPath = options.InstallerOptions.RedirectURIPath
			if receiver.installRedirectURIPath == "" {
				receiver.installRedirectURIPath = "/slack/oauth_redirect"
			}

			if options.InstallerOptions.StateVerification != nil {
				receiver.stateVerification = *options.InstallerOptions.StateVerification
			}
		} else {
			receiver.installPath = "/slack/install"
			receiver.installRedirectURIPath = "/slack/oauth_redirect"
		}

		// Create install provider
		var err error
		receiver.installer, err = oauth.NewInstallProvider(installProviderOptions)
		if err != nil {
			// Log error but don't fail - OAuth is optional
			if logger, ok := options.Logger.(*slog.Logger); ok {
				logger.Error("Failed to initialize OAuth install provider", "error", err)
			}
		}
	}

	if receiver.unhandledRequestTimeoutMillis == 0 {
		receiver.unhandledRequestTimeoutMillis = 3001
	}

	if receiver.endpoints == nil {
		receiver.endpoints = &types.ReceiverEndpoints{
			Events:      "/slack/events",
			Interactive: "/slack/events",
			Commands:    "/slack/events",
			Options:     "/slack/events",
		}
	}

	return receiver
}

// Init initializes the receiver with the app
func (r *HTTPReceiver) Init(app types.App) error {
	r.app = app
	return nil
}

// Start starts the HTTP server
func (r *HTTPReceiver) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Add default endpoints (avoid duplicates)
	registeredPaths := make(map[string]bool)

	endpoints := []string{r.endpoints.Events, r.endpoints.Interactive, r.endpoints.Commands, r.endpoints.Options}
	for _, endpoint := range endpoints {
		if endpoint != "" && !registeredPaths[endpoint] {
			mux.HandleFunc(endpoint, r.handleSlackEvent)
			registeredPaths[endpoint] = true
		}
	}

	// Add OAuth routes if installer is configured
	if r.installer != nil {
		mux.HandleFunc(r.installPath, r.handleInstallPath)
		mux.HandleFunc(r.installRedirectURIPath, r.handleInstallRedirect)
	}

	// Add custom routes
	for _, route := range r.customRoutes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	r.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		r.Stop(context.Background())
	}()

	return r.server.ListenAndServe()
}

// Stop stops the HTTP server
func (r *HTTPReceiver) Stop(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}

// handleSlackEvent handles incoming Slack events
func (r *HTTPReceiver) handleSlackEvent(w http.ResponseWriter, req *http.Request) {
	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Verify the request signature if enabled
	if r.signatureVerification {
		if err := r.verifySlackRequest(req, body); err != nil {
			http.Error(w, "Invalid request signature", http.StatusUnauthorized)
			return
		}
	}

	// Handle URL verification
	if strings.Contains(string(body), `"type":"url_verification"`) {
		r.handleURLVerification(w, body)
		return
	}

	// Handle SSL check
	if strings.Contains(string(body), `"ssl_check"`) {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Create receiver event
	headers := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	ackCalled := false
	event := types.ReceiverEvent{
		Body:    body,
		Headers: headers,
		Ack: func(response interface{}) error {
			if ackCalled {
				return errors.NewReceiverMultipleAckError()
			}
			ackCalled = true
			// Handle response body based on type
			if response == nil {
				w.WriteHeader(http.StatusOK)
			} else if responseStr, ok := response.(string); ok {
				// String response
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(responseStr))
			} else {
				// Object response - JSON encode
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				responseBytes, err := json.Marshal(response)
				if err != nil {
					// Fallback to empty response if JSON marshaling fails
					return fmt.Errorf("failed to marshal response body: %w", err)
				}
				w.Write(responseBytes)
			}
			return nil
		},
	}

	// Process the event
	ctx := req.Context()
	if err := r.app.ProcessEvent(ctx, event); err != nil {
		if !ackCalled {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Auto-ack if not already acknowledged and processBeforeResponse is false
	if !ackCalled && !r.processBeforeResponse {
		event.Ack(nil)
	}
}

// verifySlackRequest verifies the Slack request signature
func (r *HTTPReceiver) verifySlackRequest(req *http.Request, body []byte) error {
	timestamp := req.Header.Get("X-Slack-Request-Timestamp")
	signature := req.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return errors.NewReceiverAuthenticityError("Missing required headers")
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.NewReceiverAuthenticityError("Invalid timestamp")
	}

	// Check if request is too old (more than 5 minutes)
	if time.Now().Unix()-ts > 300 {
		return errors.NewReceiverAuthenticityError("Request timestamp too old")
	}

	// Create the signature base string
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	// Create HMAC
	mac := hmac.New(sha256.New, []byte(r.signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.NewReceiverAuthenticityError("Invalid signature")
	}

	return nil
}

// handleURLVerification handles Slack URL verification
func (r *HTTPReceiver) handleURLVerification(w http.ResponseWriter, body []byte) {
	// Parse the challenge from the body
	bodyStr := string(body)

	// Simple JSON parsing for challenge
	challengeStart := strings.Index(bodyStr, `"challenge":"`)
	if challengeStart == -1 {
		http.Error(w, "No challenge found", http.StatusBadRequest)
		return
	}

	challengeStart += len(`"challenge":"`)
	challengeEnd := strings.Index(bodyStr[challengeStart:], `"`)
	if challengeEnd == -1 {
		http.Error(w, "Invalid challenge format", http.StatusBadRequest)
		return
	}

	challenge := bodyStr[challengeStart : challengeStart+challengeEnd]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"challenge":"%s"}`, challenge)))
}

// handleInstallPath handles OAuth install path requests
func (r *HTTPReceiver) handleInstallPath(w http.ResponseWriter, req *http.Request) {
	if r.installer == nil {
		http.Error(w, "OAuth not configured", http.StatusNotFound)
		return
	}

	// Create install URL options
	installURLOptions := &oauth.InstallURLOptions{
		Scopes:      []string{}, // Could be configured from receiver options
		UserScopes:  []string{}, // Could be configured from receiver options
		RedirectURI: "",         // Could be configured from receiver options
	}

	// Create install path options
	installPathOptions := &oauth.InstallPathOptions{}

	// Handle the install path request
	if err := r.installer.HandleInstallPath(req, w, installPathOptions, installURLOptions); err != nil {
		if logger, ok := r.logger.(*slog.Logger); ok {
			logger.Error("Failed to handle install path request", "error", err)
		}
		http.Error(w, "Failed to handle install request", http.StatusInternalServerError)
	}
}

// handleInstallRedirect handles OAuth redirect/callback requests
func (r *HTTPReceiver) handleInstallRedirect(w http.ResponseWriter, req *http.Request) {
	if r.installer == nil {
		http.Error(w, "OAuth not configured", http.StatusNotFound)
		return
	}

	// Create callback options with default success/failure handlers
	callbackOptions := &oauth.CallbackOptions{
		Success: func(installation *oauth.Installation, installOptions *oauth.InstallURLOptions, req *http.Request, res http.ResponseWriter) {
			res.Header().Set("Content-Type", "text/html")
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`
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
</html>`))
		},
		Failure: func(err error, installOptions *oauth.InstallURLOptions, req *http.Request, res http.ResponseWriter) {
			if logger, ok := r.logger.(*slog.Logger); ok {
				logger.Error("OAuth installation failed", "error", err)
			}
			res.Header().Set("Content-Type", "text/html")
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Installation Failed</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; margin: 50px; }
        .error { color: #e01e5a; }
    </style>
</head>
<body>
    <h1 class="error">❌ Installation Failed</h1>
    <p>There was an error installing the Slack app:</p>
    <p><code>%s</code></p>
    <p>Please try again or contact support.</p>
</body>
</html>`, err.Error())))
		},
	}

	// Create install URL options (these might be retrieved from state)
	installURLOptions := &oauth.InstallURLOptions{}

	// Handle the callback request
	if err := r.installer.HandleCallback(req, w, callbackOptions, installURLOptions); err != nil {
		if logger, ok := r.logger.(*slog.Logger); ok {
			logger.Error("Failed to handle OAuth callback", "error", err)
		}
		// Error handling is done by the callback options
	}
}
