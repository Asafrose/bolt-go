package types

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/slack-go/slack/socketmode"

	"github.com/Asafrose/bolt-go/pkg/oauth"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// AckResponse represents union types for acknowledgment responses
type AckResponse interface {
	isAckResponse()
}

// Common AckResponse response types
type AckVoid struct{}
type AckString string

func (a AckVoid) isAckResponse()   {}
func (a AckString) isAckResponse() {}

// Receiver represents a receiver for handling incoming requests
type Receiver interface {
	// Init initializes the receiver
	Init(app App) error
	// Start starts the receiver
	Start(ctx context.Context) error
	// Stop stops the receiver
	Stop(ctx context.Context) error
}

// ReceiverEvent represents an event received by a receiver
type ReceiverEvent struct {
	Body        []byte                           `json:"body"`
	Headers     map[string]string                `json:"headers"`
	Ack         func(response AckResponse) error `json:"-"`
	RetryNum    int                              `json:"retry_num,omitempty"`
	RetryReason string                           `json:"retry_reason,omitempty"`
}

// App represents the main app interface that receivers need
type App interface {
	ProcessEvent(ctx context.Context, event ReceiverEvent) error
}

// HTTPReceiverOptions represents options for HTTP receiver
type HTTPReceiverOptions struct {
	SigningSecret                 string             `json:"signing_secret"`
	Logger                        *slog.Logger       `json:"logger,omitempty"`
	Endpoints                     *ReceiverEndpoints `json:"endpoints,omitempty"`
	ProcessBeforeResponse         bool               `json:"process_before_response"`
	UnhandledRequestHandler       http.HandlerFunc   `json:"-"`
	UnhandledRequestTimeoutMillis int                `json:"unhandled_request_timeout_millis"`
	CustomRoutes                  []CustomRoute      `json:"custom_routes,omitempty"`
	// Custom properties
	CustomProperties map[string]interface{} `json:"custom_properties,omitempty"`

	// OAuth configuration
	ClientID          string                  `json:"client_id,omitempty"`
	ClientSecret      string                  `json:"client_secret,omitempty"`
	StateSecret       string                  `json:"state_secret,omitempty"`
	RedirectURI       string                  `json:"redirect_uri,omitempty"`
	InstallationStore oauth.InstallationStore `json:"-"`
	Scopes            []string                `json:"scopes,omitempty"`
	InstallerOptions  *InstallerOptions       `json:"installer_options,omitempty"`
}

// ReceiverEndpoints represents custom endpoints for receivers
type ReceiverEndpoints struct {
	Events      string `json:"events"`
	Interactive string `json:"interactive"`
	Commands    string `json:"commands"`
	Options     string `json:"options"`
}

// CustomRoute represents a custom route
type CustomRoute struct {
	Path    string           `json:"path"`
	Method  string           `json:"method"`
	Handler http.HandlerFunc `json:"-"`
}

// InstallerOptions represents options for OAuth installer
type InstallerOptions struct {
	StateStore                   oauth.StateStore                                     `json:"-"`
	StateVerification            *bool                                                `json:"state_verification,omitempty"`
	LegacyStateVerification      *bool                                                `json:"legacy_state_verification,omitempty"`
	StateCookieName              string                                               `json:"state_cookie_name,omitempty"`
	StateCookieExpirationSeconds int                                                  `json:"state_cookie_expiration_seconds,omitempty"`
	AuthVersion                  string                                               `json:"auth_version,omitempty"` // v1 or v2
	DirectInstall                *bool                                                `json:"direct_install,omitempty"`
	RenderHtmlForInstallPath     func(*oauth.InstallURLOptions, *http.Request) string `json:"-"`
	InstallPath                  string                                               `json:"install_path,omitempty"`
	RedirectURIPath              string                                               `json:"redirect_uri_path,omitempty"`
	InstallPathOptions           *oauth.InstallURLOptions                             `json:"install_path_options,omitempty"`
	CallbackOptions              *oauth.CallbackOptions                               `json:"callback_options,omitempty"`
	Port                         int                                                  `json:"port,omitempty"`
	Metadata                     map[string]interface{}                               `json:"metadata,omitempty"`
	UserScopes                   []string                                             `json:"user_scopes,omitempty"`
	AuthorizationURL             string                                               `json:"authorization_url,omitempty"`
}

// SocketModeReceiverOptions represents options for Socket Mode receiver
type SocketModeReceiverOptions struct {
	AppToken                  string                                              `json:"app_token"`
	BotToken                  string                                              `json:"bot_token"`
	Logger                    *slog.Logger                                        `json:"logger,omitempty"`
	LogLevel                  LogLevel                                            `json:"log_level,omitempty"`
	PingTimeout               int                                                 `json:"ping_timeout,omitempty"`
	ClientOptions             []socketmode.Option                                 `json:"client_options,omitempty"`
	CustomProperties          map[string]interface{}                              `json:"custom_properties,omitempty"`
	CustomPropertiesExtractor func(map[string]interface{}) map[string]interface{} `json:"-"`
	CustomRoutes              []CustomRoute                                       `json:"custom_routes,omitempty"`

	// OAuth configuration
	ClientID          string                  `json:"client_id,omitempty"`
	ClientSecret      string                  `json:"client_secret,omitempty"`
	StateSecret       string                  `json:"state_secret,omitempty"`
	RedirectURI       string                  `json:"redirect_uri,omitempty"`
	InstallationStore oauth.InstallationStore `json:"-"`
	Scopes            []string                `json:"scopes,omitempty"`
	InstallerOptions  *InstallerOptions       `json:"installer_options,omitempty"`
}

// AwsLambdaReceiverOptions represents options for AWS Lambda receiver
type AwsLambdaReceiverOptions struct {
	SigningSecret         string                 `json:"signing_secret"`
	Logger                *slog.Logger           `json:"logger,omitempty"`
	LogLevel              LogLevel               `json:"log_level,omitempty"`
	ProcessBeforeResponse bool                   `json:"process_before_response"`
	SignatureVerification *bool                  `json:"signature_verification,omitempty"`
	CustomProperties      map[string]interface{} `json:"custom_properties,omitempty"`
}
