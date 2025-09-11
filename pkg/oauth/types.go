package oauth

import (
	"context"
	"net/http"
	"time"
)

// InstallationStore interface for storing and retrieving installations
type InstallationStore interface {
	StoreInstallation(ctx context.Context, installation *Installation) error
	FetchInstallation(ctx context.Context, installQuery InstallationQuery) (*Installation, error)
	DeleteInstallation(ctx context.Context, installQuery InstallationQuery) error
}

// StateStore interface for managing OAuth state
type StateStore interface {
	GenerateStateParam(ctx context.Context, installOptions *InstallURLOptions) (string, error)
	VerifyStateParam(ctx context.Context, state string) (*InstallURLOptions, error)
}

// Installation represents a Slack app installation
type Installation struct {
	Team                *Team                  `json:"team,omitempty"`
	Enterprise          *Enterprise            `json:"enterprise,omitempty"`
	User                *User                  `json:"user,omitempty"`
	TokenType           string                 `json:"token_type,omitempty"`
	IsEnterpriseInstall bool                   `json:"is_enterprise_install"`
	AppID               string                 `json:"app_id,omitempty"`
	AuthVersion         string                 `json:"auth_version,omitempty"`
	Bot                 *Bot                   `json:"bot,omitempty"`
	IncomingWebhook     *IncomingWebhook       `json:"incoming_webhook,omitempty"`
	AuthedUser          *AuthedUser            `json:"authed_user,omitempty"`
	Scope               string                 `json:"scope,omitempty"`
	TokenType2          string                 `json:"token_type_2,omitempty"`
	AccessToken         string                 `json:"access_token,omitempty"`
	BotToken            string                 `json:"bot_token,omitempty"`
	BotID               string                 `json:"bot_id,omitempty"`
	BotUserID           string                 `json:"bot_user_id,omitempty"`
	BotScopes           []string               `json:"bot_scopes,omitempty"`
	UserScopes          []string               `json:"user_scopes,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// Team represents a Slack team/workspace
type Team struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// Enterprise represents a Slack enterprise
type Enterprise struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// User represents a Slack user
type User struct {
	ID           string     `json:"id"`
	Name         string     `json:"name,omitempty"`
	Email        string     `json:"email,omitempty"`
	TeamID       string     `json:"team_id,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	Scope        string     `json:"scope,omitempty"`
}

// Bot represents a Slack bot
type Bot struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	Scope        string     `json:"scope,omitempty"`
}

// IncomingWebhook represents a Slack incoming webhook
type IncomingWebhook struct {
	Channel          string `json:"channel,omitempty"`
	ChannelID        string `json:"channel_id,omitempty"`
	ConfigurationURL string `json:"configuration_url,omitempty"`
	URL              string `json:"url,omitempty"`
}

// AuthedUser represents an authenticated user
type AuthedUser struct {
	ID           string     `json:"id"`
	Scope        string     `json:"scope,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
}

// InstallationQuery represents a query for retrieving installations
type InstallationQuery struct {
	TeamID              string `json:"team_id,omitempty"`
	EnterpriseID        string `json:"enterprise_id,omitempty"`
	UserID              string `json:"user_id,omitempty"`
	ConversationID      string `json:"conversation_id,omitempty"`
	IsEnterpriseInstall bool   `json:"is_enterprise_install"`
}

// InstallURLOptions represents options for generating install URLs
type InstallURLOptions struct {
	Scopes      []string               `json:"scopes,omitempty"`
	UserScopes  []string               `json:"user_scopes,omitempty"`
	RedirectURI string                 `json:"redirect_uri,omitempty"`
	TeamID      string                 `json:"team_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// InstallPathOptions represents options for the install path
type InstallPathOptions struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CallbackOptions represents options for OAuth callbacks
type CallbackOptions struct {
	Success func(installation *Installation, installOptions *InstallURLOptions, req *http.Request, res http.ResponseWriter)
	Failure func(error error, installOptions *InstallURLOptions, req *http.Request, res http.ResponseWriter)
}

// InstallProviderOptions represents options for the OAuth install provider
type InstallProviderOptions struct {
	ClientID                     string                                         `json:"client_id"`
	ClientSecret                 string                                         `json:"client_secret"`
	StateSecret                  string                                         `json:"state_secret,omitempty"`
	InstallationStore            InstallationStore                              `json:"-"`
	AuthVersion                  string                                         `json:"auth_version,omitempty"` // v1 or v2, default v2
	Logger                       interface{}                                    `json:"logger,omitempty"`
	LogLevel                     interface{}                                    `json:"log_level,omitempty"`
	StateStore                   StateStore                                     `json:"-"`
	StateVerification            *bool                                          `json:"state_verification,omitempty"` // default true
	LegacyStateVerification      *bool                                          `json:"legacy_state_verification,omitempty"`
	StateCookieName              string                                         `json:"state_cookie_name,omitempty"`
	StateCookieExpirationSeconds int                                            `json:"state_cookie_expiration_seconds,omitempty"`
	DirectInstall                *bool                                          `json:"direct_install,omitempty"`
	RenderHtmlForInstallPath     func(*InstallURLOptions, *http.Request) string `json:"-"`
	AuthorizationURL             string                                         `json:"authorization_url,omitempty"`
}

// OAuthV2Response represents the response from OAuth v2 access endpoint
type OAuthV2Response struct {
	OK                  bool             `json:"ok"`
	Error               string           `json:"error,omitempty"`
	AccessToken         string           `json:"access_token,omitempty"`
	TokenType           string           `json:"token_type,omitempty"`
	Scope               string           `json:"scope,omitempty"`
	BotUserID           string           `json:"bot_user_id,omitempty"`
	AppID               string           `json:"app_id,omitempty"`
	Team                *Team            `json:"team,omitempty"`
	Enterprise          *Enterprise      `json:"enterprise,omitempty"`
	AuthedUser          *AuthedUser      `json:"authed_user,omitempty"`
	IsEnterpriseInstall bool             `json:"is_enterprise_install"`
	IncomingWebhook     *IncomingWebhook `json:"incoming_webhook,omitempty"`
	RefreshToken        string           `json:"refresh_token,omitempty"`
	ExpiresIn           int              `json:"expires_in,omitempty"`
}

// OAuthV1Response represents the response from OAuth v1 access endpoint
type OAuthV1Response struct {
	OK              bool             `json:"ok"`
	Error           string           `json:"error,omitempty"`
	AccessToken     string           `json:"access_token,omitempty"`
	Scope           string           `json:"scope,omitempty"`
	UserID          string           `json:"user_id,omitempty"`
	TeamID          string           `json:"team_id,omitempty"`
	TeamName        string           `json:"team_name,omitempty"`
	Bot             *Bot             `json:"bot,omitempty"`
	IncomingWebhook *IncomingWebhook `json:"incoming_webhook,omitempty"`
}
