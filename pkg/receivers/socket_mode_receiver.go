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
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SocketModeReceiver handles Socket Mode connections from Slack using the official socketmode client
type SocketModeReceiver struct {
	appToken                  string
	logger                    *slog.Logger
	client                    *socketmode.Client
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

	app    types.App
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSocketModeReceiver creates a new Socket Mode receiver
func NewSocketModeReceiver(options types.SocketModeReceiverOptions) *SocketModeReceiver {
	// Create slack API client
	slackClient := slack.New(options.AppToken)

	// Create socketmode client options
	socketmodeOptions := []socketmode.Option{}

	// Add ping interval if specified
	if options.PingTimeout > 0 {
		pingInterval := time.Duration(options.PingTimeout) * time.Millisecond
		socketmodeOptions = append(socketmodeOptions, socketmode.OptionPingInterval(pingInterval))
	}

	// Create socketmode client
	client := socketmode.New(slackClient, socketmodeOptions...)

	receiver := &SocketModeReceiver{
		appToken:                  options.AppToken,
		logger:                    options.Logger,
		client:                    client,
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

		// Set installation store if provided
		if options.InstallationStore != nil {
			installProviderOptions.InstallationStore = options.InstallationStore
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

	// Set logger
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

	// Set up event handling
	r.setupEventHandlers()

	// Start the socketmode client
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if err := r.client.RunContext(r.ctx); err != nil {
			r.logger.Error("Socket mode client error", "error", err)
		}
	}()

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

// setupEventHandlers configures event handlers for the socketmode client
func (r *SocketModeReceiver) setupEventHandlers() {
	// Handle all socketmode events
	go func() {
		for evt := range r.client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				r.logger.Info("Connecting to Slack with Socket Mode")
			case socketmode.EventTypeConnectionError:
				r.logger.Error("Connection failed", "error", evt.Data)
			case socketmode.EventTypeConnected:
				r.logger.Info("Connected to Slack with Socket Mode")
			case socketmode.EventTypeEventsAPI:
				r.handleEventsAPI(evt)
			case socketmode.EventTypeInteractive:
				r.handleInteractive(evt)
			case socketmode.EventTypeSlashCommand:
				r.handleSlashCommand(evt)
			case socketmode.EventTypeHello:
				r.logger.Info("Received hello message from Slack")
			case socketmode.EventTypeDisconnect:
				r.logger.Info("Received disconnect message from Slack")
			default:
				r.logger.Warn("Received unknown event type", "type", evt.Type)
			}
		}
	}()
}

// handleEventsAPI handles Events API messages
func (r *SocketModeReceiver) handleEventsAPI(evt socketmode.Event) {
	r.processEvent(evt)
}

// handleInteractive handles interactive messages
func (r *SocketModeReceiver) handleInteractive(evt socketmode.Event) {
	r.processEvent(evt)
}

// handleSlashCommand handles slash command messages
func (r *SocketModeReceiver) handleSlashCommand(evt socketmode.Event) {
	r.processEvent(evt)
}

// processEvent processes an event through the app
func (r *SocketModeReceiver) processEvent(evt socketmode.Event) {
	// The request is directly available in the event
	req := evt.Request
	if req == nil {
		r.logger.Error("No request in socket mode event")
		return
	}

	// Convert payload to JSON bytes
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		r.logger.Error("Failed to marshal payload", "error", err)
		return
	}

	// Create headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	ackCalled := false
	event := types.ReceiverEvent{
		Body:    payloadBytes,
		Headers: headers,
		Ack: func(response types.AckResponse) error {
			if ackCalled {
				return errors.NewReceiverMultipleAckError()
			}
			ackCalled = true

			// Send acknowledgment back to Slack using the official client
			r.client.Ack(*req, response)
			return nil
		},
	}

	// Add custom properties if extractor is provided
	if r.customPropertiesExtractor != nil {
		// Convert request to map for custom properties extraction
		reqMap := map[string]interface{}{
			"type":                     req.Type,
			"envelope_id":              req.EnvelopeID,
			"payload":                  req.Payload,
			"accepts_response_payload": req.AcceptsResponsePayload,
			"retry_attempt":            req.RetryAttempt,
			"retry_reason":             req.RetryReason,
		}
		customProps := r.customPropertiesExtractor(reqMap)
		// Note: We could extend ReceiverEvent to include custom properties if needed
		_ = customProps
	}

	// Process the event
	if err := r.app.ProcessEvent(r.ctx, event); err != nil {
		r.logger.Error("Failed to process event", "error", err)
		if !ackCalled {
			if ackErr := event.Ack(nil); ackErr != nil {
				// Log error but don't fail the request
				r.logger.Error("Failed to ack event", "error", ackErr)
			}
		}
		return
	}

	// Auto-acknowledge if not already done
	if !ackCalled {
		if err := event.Ack(nil); err != nil {
			r.logger.Error("Failed to auto-ack event", "error", err)
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
		Addr:              fmt.Sprintf(":%d", r.httpServerPort),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
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
			if _, err := res.Write([]byte(`
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
</html>`)); err != nil {
				// Error already sent to client, just log it
				_ = err
			}
		},
		Failure: func(err error, installOptions *oauth.InstallURLOptions, req *http.Request, res http.ResponseWriter) {
			r.logger.Error("OAuth installation failed", "error", err)
			res.Header().Set("Content-Type", "text/html")
			res.WriteHeader(http.StatusBadRequest)
			if _, writeErr := res.Write([]byte(fmt.Sprintf(`
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
</html>`, err.Error()))); writeErr != nil {
				// Error already sent to client, just log it
				_ = writeErr
			}
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

// cleanup closes the socketmode client and HTTP server
func (r *SocketModeReceiver) cleanup() {
	// The socketmode client will be closed when the context is cancelled
	// No need to explicitly close it here

	if r.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := r.httpServer.Shutdown(ctx); err != nil {
			// Log error but don't fail cleanup
			r.logger.Error("Failed to shutdown HTTP server", "error", err)
		}
	}
}
