package receivers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/gorilla/websocket"
	"github.com/slack-go/slack"
)

// SocketModeReceiver handles Socket Mode connections from Slack
type SocketModeReceiver struct {
	appToken                  string
	logger                    *slog.Logger
	logLevel                  interface{}
	pingTimeout               time.Duration
	customProperties          map[string]interface{}
	customPropertiesExtractor func(map[string]interface{}) map[string]interface{}
	customRoutes              []types.CustomRoute

	// OAuth support
	installer              *oauth.InstallProvider
	httpServer             *http.Server
	httpServerPort         int
	installPath            string
	installRedirectURIPath string
	stateVerification      bool

	conn   *websocket.Conn
	app    types.App
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SocketModeMessage represents a Socket Mode message
type SocketModeMessage struct {
	Type                   string                 `json:"type"`
	EnvelopeID             string                 `json:"envelope_id,omitempty"`
	Payload                map[string]interface{} `json:"payload,omitempty"`
	AcceptsResponsePayload bool                   `json:"accepts_response_payload,omitempty"`
}

// SocketModeAck represents an acknowledgment message
type SocketModeAck struct {
	EnvelopeID string      `json:"envelope_id"`
	Payload    interface{} `json:"payload,omitempty"`
}

// NewSocketModeReceiver creates a new Socket Mode receiver
func NewSocketModeReceiver(options types.SocketModeReceiverOptions) *SocketModeReceiver {
	receiver := &SocketModeReceiver{
		appToken:                  options.AppToken,
		pingTimeout:               30 * time.Second,
		customProperties:          options.CustomProperties,
		customPropertiesExtractor: options.CustomPropertiesExtractor,
		customRoutes:              options.CustomRoutes,
		stateVerification:         true, // default to true
		httpServerPort:            3000, // default port
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

			// Set HTTP server port
			if options.InstallerOptions.Port > 0 {
				receiver.httpServerPort = options.InstallerOptions.Port
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
			if receiver.logger != nil {
				receiver.logger.Error("Failed to initialize OAuth install provider", "error", err)
			}
		}
	}

	if options.PingTimeout > 0 {
		receiver.pingTimeout = time.Duration(options.PingTimeout) * time.Millisecond
	}

	if receiver.logger == nil {
		receiver.logger = slog.Default()
	}

	return receiver
}

// Init initializes the receiver with the app
func (r *SocketModeReceiver) Init(app types.App) error {
	r.app = app
	return nil
}

// Start starts the Socket Mode connection
func (r *SocketModeReceiver) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start HTTP server if OAuth is configured or custom routes are provided
	if r.installer != nil || len(r.customRoutes) > 0 {
		if err := r.startHTTPServer(); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}

	// Get WebSocket URL from Slack
	wsURL, err := r.getWebSocketURL()
	if err != nil {
		return fmt.Errorf("failed to get WebSocket URL: %w", err)
	}

	// Connect to WebSocket
	if err := r.connect(wsURL); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Start message handling goroutines
	r.wg.Add(2)
	go r.readMessages()
	go r.pingLoop()

	// Wait for context cancellation
	<-r.ctx.Done()

	// Clean up
	r.cleanup()
	r.wg.Wait()

	return nil
}

// Stop stops the Socket Mode connection
func (r *SocketModeReceiver) Stop(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// getWebSocketURL retrieves the WebSocket URL from Slack using the slack SDK
func (r *SocketModeReceiver) getWebSocketURL() (string, error) {
	// Create a slack client with the app token
	client := slack.New(r.appToken)

	// Use the slack SDK to start socket mode connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, websocketURL, err := client.StartSocketModeContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get socket mode URL: %w", err)
	}

	return websocketURL, nil
}

// connect establishes the WebSocket connection
func (r *SocketModeReceiver) connect(wsURL string) error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	r.conn = conn
	return nil
}

// readMessages reads messages from the WebSocket connection
func (r *SocketModeReceiver) readMessages() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		var msg SocketModeMessage
		if err := r.conn.ReadJSON(&msg); err != nil {
			r.logger.Error("Failed to read WebSocket message", "error", err)
			return
		}

		if err := r.handleMessage(msg); err != nil {
			r.logger.Error("Failed to handle message", "error", err)
		}
	}
}

// handleMessage handles incoming Socket Mode messages
func (r *SocketModeReceiver) handleMessage(msg SocketModeMessage) error {
	switch msg.Type {
	case "hello":
		r.logger.Info("Received hello message from Slack")
		return nil

	case "disconnect":
		r.logger.Info("Received disconnect message from Slack")
		r.cancel()
		return nil

	case "events_api":
		return r.handleEventsAPI(msg)

	case "interactive":
		return r.handleInteractive(msg)

	case "slash_commands":
		return r.handleSlashCommand(msg)

	case "options_request":
		return r.handleOptionsRequest(msg)

	default:
		r.logger.Warn("Received unknown message type", "type", msg.Type)
		return nil
	}
}

// handleEventsAPI handles Events API messages
func (r *SocketModeReceiver) handleEventsAPI(msg SocketModeMessage) error {
	return r.processEvent(msg)
}

// handleInteractive handles interactive messages
func (r *SocketModeReceiver) handleInteractive(msg SocketModeMessage) error {
	return r.processEvent(msg)
}

// handleSlashCommand handles slash command messages
func (r *SocketModeReceiver) handleSlashCommand(msg SocketModeMessage) error {
	return r.processEvent(msg)
}

// handleOptionsRequest handles options request messages
func (r *SocketModeReceiver) handleOptionsRequest(msg SocketModeMessage) error {
	return r.processEvent(msg)
}

// processEvent processes an event through the app
func (r *SocketModeReceiver) processEvent(msg SocketModeMessage) error {
	// Convert payload to JSON bytes
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}

	// Create headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	ackCalled := false
	event := types.ReceiverEvent{
		Body:    payloadBytes,
		Headers: headers,
		Ack: func(response interface{}) error {
			if ackCalled {
				return errors.NewReceiverMultipleAckError()
			}
			ackCalled = true

			// Send acknowledgment back to Slack
			ack := SocketModeAck{
				EnvelopeID: msg.EnvelopeID,
				Payload:    response,
			}
			return r.conn.WriteJSON(ack)
		},
	}

	// Process the event
	if err := r.app.ProcessEvent(r.ctx, event); err != nil {
		r.logger.Error("Failed to process event", "error", err)
		if !ackCalled {
			event.Ack(nil) // Still acknowledge to avoid retries
		}
		return err
	}

	// Auto-acknowledge if not already done
	if !ackCalled {
		return event.Ack(nil)
	}

	return nil
}

// pingLoop sends periodic ping messages to keep the connection alive
func (r *SocketModeReceiver) pingLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.pingTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if err := r.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				r.logger.Error("Failed to send ping", "error", err)
				return
			}
		}
	}
}

// startHTTPServer starts the HTTP server for OAuth and custom routes
func (r *SocketModeReceiver) startHTTPServer() error {
	mux := http.NewServeMux()

	// Add OAuth routes if installer is configured
	if r.installer != nil {
		mux.HandleFunc(r.installPath, r.handleInstallPath)
		mux.HandleFunc(r.installRedirectURIPath, r.handleInstallRedirect)
	}

	// Add custom routes
	for _, route := range r.customRoutes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	r.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.httpServerPort),
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// handleInstallPath handles OAuth install path requests
func (r *SocketModeReceiver) handleInstallPath(w http.ResponseWriter, req *http.Request) {
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
		r.logger.Error("Failed to handle install path request", "error", err)
		http.Error(w, "Failed to handle install request", http.StatusInternalServerError)
	}
}

// handleInstallRedirect handles OAuth redirect/callback requests
func (r *SocketModeReceiver) handleInstallRedirect(w http.ResponseWriter, req *http.Request) {
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
			r.logger.Error("OAuth installation failed", "error", err)
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
		r.logger.Error("Failed to handle OAuth callback", "error", err)
		// Error handling is done by the callback options
	}
}

// cleanup closes the WebSocket connection and HTTP server
func (r *SocketModeReceiver) cleanup() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.httpServer.Shutdown(ctx)
	}
}
