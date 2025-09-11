package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Asafrose/bolt-go/pkg/conversation"
	bolterrors "github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/middleware"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// AppOptions represents configuration options for the App
type AppOptions struct {
	// Receiver configuration
	SigningSecret         *string                  `json:"signing_secret,omitempty"`
	Endpoints             *types.ReceiverEndpoints `json:"endpoints,omitempty"`
	Port                  *int                     `json:"port,omitempty"`
	CustomRoutes          []types.CustomRoute      `json:"custom_routes,omitempty"`
	ProcessBeforeResponse bool                     `json:"process_before_response"`
	SignatureVerification bool                     `json:"signature_verification"`

	// OAuth configuration
	ClientID          *string     `json:"client_id,omitempty"`
	ClientSecret      *string     `json:"client_secret,omitempty"`
	StateSecret       *string     `json:"state_secret,omitempty"`
	RedirectURI       *string     `json:"redirect_uri,omitempty"`
	InstallationStore interface{} `json:"installation_store,omitempty"`
	Scopes            []string    `json:"scopes,omitempty"`
	InstallerOptions  interface{} `json:"installer_options,omitempty"`

	// Client configuration
	HTTPClient    *http.Client   `json:"-"`
	ClientOptions []slack.Option `json:"-"`
	Token         *string        `json:"token,omitempty"`
	AppToken      *string        `json:"app_token,omitempty"`
	BotID         *string        `json:"bot_id,omitempty"`
	BotUserID     *string        `json:"bot_user_id,omitempty"`

	// Authorization
	Authorize AuthorizeFunc `json:"-"`

	// Receiver
	Receiver types.Receiver `json:"-"`

	// Logging
	Logger   *slog.Logger `json:"-"`
	LogLevel *LogLevel    `json:"log_level,omitempty"`

	// Behavior
	IgnoreSelf               *bool `json:"ignore_self,omitempty"`
	SocketMode               bool  `json:"socket_mode"`
	DeveloperMode            bool  `json:"developer_mode"`
	TokenVerificationEnabled bool  `json:"token_verification_enabled"`
	DeferInitialization      bool  `json:"defer_initialization"`
	ExtendedErrorHandler     bool  `json:"extended_error_handler"`
	AttachFunctionToken      bool  `json:"attach_function_token"`

	// Conversation store
	ConvoStore conversation.ConversationStore `json:"convo_store,omitempty"`
}

// AuthorizeSourceData represents data provided to authorization function
type AuthorizeSourceData struct {
	TeamID              *string `json:"team_id,omitempty"`
	EnterpriseID        *string `json:"enterprise_id,omitempty"`
	UserID              *string `json:"user_id,omitempty"`
	ConversationID      *string `json:"conversation_id,omitempty"`
	IsEnterpriseInstall bool    `json:"is_enterprise_install"`
}

// AuthorizeResult represents the result of authorization
type AuthorizeResult struct {
	BotToken     *string                `json:"bot_token,omitempty"`
	UserToken    *string                `json:"user_token,omitempty"`
	BotID        *string                `json:"bot_id,omitempty"`
	BotUserID    *string                `json:"bot_user_id,omitempty"`
	UserID       *string                `json:"user_id,omitempty"`
	TeamID       *string                `json:"team_id,omitempty"`
	EnterpriseID *string                `json:"enterprise_id,omitempty"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// AuthorizeFunc represents an authorization function
type AuthorizeFunc func(ctx context.Context, source AuthorizeSourceData, body interface{}) (*AuthorizeResult, error)

// ErrorHandler represents an error handler function
type ErrorHandler func(err error) error

// ExtendedErrorHandler represents an extended error handler function
type ExtendedErrorHandler func(ctx context.Context, err error, logger *slog.Logger, body interface{}, context *types.Context) error

// listenerConstraints holds the matching constraints for a listener
type listenerConstraints struct {
	eventType      *string
	messagePattern interface{}
	actionID       *string
	blockID        *string
	callbackID     *string
	command        *string
	shortcutType   *string
	viewType       *string
	actionType     *string // For action type constraints (e.g., "block_actions")
	// RegExp patterns
	actionIDPattern   *regexp.Regexp
	blockIDPattern    *regexp.Regexp
	callbackIDPattern *regexp.Regexp
	commandPattern    *regexp.Regexp
	eventTypePattern  *regexp.Regexp
}

// listenerEntry represents a registered listener with its constraints
type listenerEntry struct {
	eventType   helpers.IncomingEventType
	constraints listenerConstraints
	middleware  []types.Middleware[types.AllMiddlewareArgs]
}

// WebClientPool manages a pool of Slack clients
type WebClientPool struct {
	mu      sync.RWMutex
	clients map[string]*slack.Client
}

// NewWebClientPool creates a new WebClientPool
func NewWebClientPool() *WebClientPool {
	return &WebClientPool{
		clients: make(map[string]*slack.Client),
	}
}

// GetOrCreate gets or creates a client for the given token
func (p *WebClientPool) GetOrCreate(token string, options ...slack.Option) *slack.Client {
	p.mu.RLock()
	client, exists := p.clients[token]
	p.mu.RUnlock()

	if exists {
		return client
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[token]; exists {
		return client
	}

	client = slack.New(token, options...)
	p.clients[token] = client
	return client
}

// App represents a Slack app
type App struct {
	// Public fields
	Client *slack.Client
	Logger *slog.Logger

	// Private fields
	clientOptions            []slack.Option
	clients                  map[string]*WebClientPool
	receiver                 types.Receiver
	logLevel                 LogLevel
	authorize                AuthorizeFunc
	middleware               []types.Middleware[types.AllMiddlewareArgs]
	listeners                [][]types.Middleware[types.AllMiddlewareArgs] // Deprecated
	listenerEntries          []*listenerEntry
	errorHandler             interface{} // ErrorHandler or ExtendedErrorHandler
	socketMode               bool
	developerMode            bool
	extendedErrorHandler     bool
	hasCustomErrorHandler    bool
	tokenVerificationEnabled bool
	initialized              bool
	attachFunctionToken      bool
	conversationStore        conversation.ConversationStore

	// Used when defer initialization is true
	argToken         *string
	argAuthorize     AuthorizeFunc
	argAuthorization *AuthorizeResult

	mu sync.RWMutex
}

// New creates a new Slack App
func New(options AppOptions) (*App, error) {
	// Validate conflicting options
	if options.Token != nil && options.Authorize != nil {
		return nil, errors.New("cannot specify both token and authorize callback")
	}

	if options.SocketMode && options.Receiver != nil {
		return nil, errors.New("cannot specify both socketMode and custom receiver")
	}

	app := &App{
		middleware:               make([]types.Middleware[types.AllMiddlewareArgs], 0),
		listeners:                make([][]types.Middleware[types.AllMiddlewareArgs], 0),
		clients:                  make(map[string]*WebClientPool),
		developerMode:            options.DeveloperMode,
		socketMode:               options.SocketMode,
		tokenVerificationEnabled: options.TokenVerificationEnabled,
		extendedErrorHandler:     options.ExtendedErrorHandler,
		attachFunctionToken:      options.AttachFunctionToken,
	}

	// Set up logging
	if options.Logger != nil {
		app.Logger = options.Logger
	} else {
		app.Logger = slog.Default()
	}

	if options.LogLevel != nil {
		app.logLevel = *options.LogLevel
	} else if options.DeveloperMode {
		app.logLevel = LogLevelDebug
	} else {
		app.logLevel = LogLevelInfo
	}

	// Set up client options
	app.clientOptions = []slack.Option{}
	if options.ClientOptions != nil {
		app.clientOptions = append(app.clientOptions, options.ClientOptions...)
	}

	// Create the main client
	if options.Token != nil {
		app.Client = slack.New(*options.Token, app.clientOptions...)
	} else {
		app.Client = slack.New("", app.clientOptions...)
	}

	// Set up error handler
	app.errorHandler = app.defaultErrorHandler
	app.hasCustomErrorHandler = false

	// Set up receiver
	if options.Receiver != nil {
		app.receiver = options.Receiver
	} else {
		// Create default receiver based on options
		receiver, err := app.initReceiver(options)
		if err != nil {
			return nil, err
		}
		app.receiver = receiver
	}

	// Set up authorization
	if options.DeferInitialization {
		app.argToken = options.Token
		app.argAuthorize = options.Authorize
		if options.Token != nil {
			app.argAuthorization = &AuthorizeResult{
				BotID:     options.BotID,
				BotUserID: options.BotUserID,
				BotToken:  options.Token,
			}
		}
		app.initialized = false
	} else {
		authorize, err := app.initAuthorize(options.Token, options.Authorize, options.BotID, options.BotUserID)
		if err != nil {
			return nil, err
		}
		app.authorize = authorize
		app.initialized = true
	}

	// Add ignore self middleware (enabled by default, can be disabled by setting IgnoreSelf to false)
	ignoreSelfEnabled := true // Default to true like in Bolt-JS
	if options.IgnoreSelf != nil && !*options.IgnoreSelf {
		ignoreSelfEnabled = false // Only disable if explicitly set to false
	}

	if ignoreSelfEnabled {
		app.Use(middleware.IgnoreSelf())
	}

	// Initialize conversation store if not provided
	if options.ConvoStore != nil {
		app.conversationStore = options.ConvoStore
	} else {
		// Use default MemoryStore
		app.conversationStore = conversation.NewMemoryStore()
	}

	// Add conversation middleware to provide conversation context
	if app.conversationStore != nil {
		app.Use(conversation.ConversationContext(app.conversationStore))
	}

	// Initialize receiver
	if err := app.receiver.Init(app); err != nil {
		return nil, err
	}

	return app, nil
}

// Init initializes the app if defer initialization was used
func (a *App) Init(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	authorize, err := a.initAuthorize(a.argToken, a.argAuthorize, nil, nil)
	if err != nil {
		return err
	}

	a.authorize = authorize
	a.initialized = true
	return nil
}

// Use registers global middleware
func (a *App) Use(middleware types.Middleware[types.AllMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.middleware = append(a.middleware, middleware)
	return a
}

// Event registers event listeners
func (a *App) Event(eventType string, middleware ...types.Middleware[types.SlackEventMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry with routing information
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeEvent,
		constraints: listenerConstraints{
			eventType: &eventType,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert event middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapEventMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// EventPattern adds a listener for events matching a regular expression pattern
func (a *App) EventPattern(pattern *regexp.Regexp, middleware ...types.Middleware[types.SlackEventMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for events with RegExp pattern
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeEvent,
		constraints: listenerConstraints{
			eventTypePattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert event middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapEventMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Message registers message listeners
func (a *App) Message(pattern interface{}, middleware ...types.Middleware[types.SlackEventMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for message events
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeEvent,
		constraints: listenerConstraints{
			eventType:      stringPtr("message"),
			messagePattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert event middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapEventMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Action registers action listeners
func (a *App) Action(constraints types.ActionConstraints, middleware ...types.Middleware[types.SlackActionMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for actions
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeAction,
		constraints: listenerConstraints{
			actionID:          constraints.ActionID,
			blockID:           constraints.BlockID,
			callbackID:        constraints.CallbackID,
			actionType:        constraints.Type,
			actionIDPattern:   constraints.ActionIDPattern,
			blockIDPattern:    constraints.BlockIDPattern,
			callbackIDPattern: constraints.CallbackIDPattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert action middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapActionMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Command registers command listeners
func (a *App) Command(command string, middleware ...types.Middleware[types.SlackCommandMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for commands
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeCommand,
		constraints: listenerConstraints{
			command: &command,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert command middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapCommandMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// CommandPattern adds a listener for commands matching a regular expression pattern
func (a *App) CommandPattern(pattern *regexp.Regexp, middleware ...types.Middleware[types.SlackCommandMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for commands with RegExp pattern
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeCommand,
		constraints: listenerConstraints{
			commandPattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert command middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapCommandMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Shortcut registers shortcut listeners
func (a *App) Shortcut(constraints types.ShortcutConstraints, middleware ...types.Middleware[types.SlackShortcutMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for shortcuts
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeShortcut,
		constraints: listenerConstraints{
			callbackID:   constraints.CallbackID,
			shortcutType: constraints.Type,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert shortcut middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapShortcutMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// ShortcutString adds a listener for shortcuts matching a callback ID string
func (a *App) ShortcutString(callbackID string, middleware ...types.Middleware[types.SlackShortcutMiddlewareArgs]) *App {
	return a.Shortcut(types.ShortcutConstraints{
		CallbackID: &callbackID,
	}, middleware...)
}

// ShortcutPattern adds a listener for shortcuts matching a callback ID RegExp pattern
func (a *App) ShortcutPattern(pattern *regexp.Regexp, middleware ...types.Middleware[types.SlackShortcutMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for shortcuts with RegExp pattern
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeShortcut,
		constraints: listenerConstraints{
			callbackIDPattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert shortcut middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapShortcutMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// View registers view listeners
func (a *App) View(constraints types.ViewConstraints, middleware ...types.Middleware[types.SlackViewMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for views
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeViewAction,
		constraints: listenerConstraints{
			callbackID: constraints.CallbackID,
			viewType:   constraints.Type,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert view middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapViewMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// ViewString adds a listener for views matching a callback ID string
func (a *App) ViewString(callbackID string, middleware ...types.Middleware[types.SlackViewMiddlewareArgs]) *App {
	return a.View(types.ViewConstraints{
		CallbackID: &callbackID,
	}, middleware...)
}

// ViewPattern adds a listener for views matching a callback ID RegExp pattern
func (a *App) ViewPattern(pattern *regexp.Regexp, middleware ...types.Middleware[types.SlackViewMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for views with RegExp pattern
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeViewAction,
		constraints: listenerConstraints{
			callbackIDPattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert view middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapViewMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Options registers options listeners
func (a *App) Options(constraints types.OptionsConstraints, middleware ...types.Middleware[types.SlackOptionsMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for options
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeOptions,
		constraints: listenerConstraints{
			actionID: constraints.ActionID,
			blockID:  constraints.BlockID,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert options middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapOptionsMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// OptionsString adds a listener for options matching an action ID string
func (a *App) OptionsString(actionID string, middleware ...types.Middleware[types.SlackOptionsMiddlewareArgs]) *App {
	return a.Options(types.OptionsConstraints{
		ActionID: &actionID,
	}, middleware...)
}

// OptionsPattern adds a listener for options matching an action ID RegExp pattern
func (a *App) OptionsPattern(pattern *regexp.Regexp, middleware ...types.Middleware[types.SlackOptionsMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create a listener entry for options with RegExp pattern
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeOptions,
		constraints: listenerConstraints{
			actionIDPattern: pattern,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Convert options middleware to base middleware
	for _, m := range middleware {
		listener.middleware = append(listener.middleware, a.wrapOptionsMiddleware(m))
	}

	a.listenerEntries = append(a.listenerEntries, listener)
	return a
}

// Assistant registers an assistant for handling AI assistant events
func (a *App) Assistant(assistant interface{}) *App {
	// The assistant should implement a GetMiddleware() method
	if assistantWithMiddleware, ok := assistant.(interface {
		GetMiddleware() types.Middleware[types.AllMiddlewareArgs]
	}); ok {
		middleware := assistantWithMiddleware.GetMiddleware()
		a.Use(middleware)
	}
	return a
}

// Function registers a custom function handler
func (a *App) Function(callbackID string, middleware ...interface{}) *App {
	// Handle different parameter patterns:
	// Function(callbackID, handler)
	// Function(callbackID, options, handler)

	var options *types.CustomFunctionOptions
	var handler types.Middleware[types.SlackCustomFunctionMiddlewareArgs]

	if len(middleware) == 1 {
		// Function(callbackID, handler)
		// Use reflection to check if it's a function with the right signature
		if h, ok := middleware[0].(func(types.SlackCustomFunctionMiddlewareArgs) error); ok {
			handler = types.Middleware[types.SlackCustomFunctionMiddlewareArgs](h)
			options = &types.CustomFunctionOptions{AutoAcknowledge: true} // Default
		} else if h, ok := middleware[0].(types.Middleware[types.SlackCustomFunctionMiddlewareArgs]); ok {
			handler = h
			options = &types.CustomFunctionOptions{AutoAcknowledge: true} // Default
		}
	} else if len(middleware) == 2 {
		// Function(callbackID, options, handler)
		if opts, ok := middleware[0].(types.CustomFunctionOptions); ok {
			options = &opts
		} else if opts, ok := middleware[0].(*types.CustomFunctionOptions); ok {
			options = opts
		}
		if h, ok := middleware[1].(func(types.SlackCustomFunctionMiddlewareArgs) error); ok {
			handler = types.Middleware[types.SlackCustomFunctionMiddlewareArgs](h)
		} else if h, ok := middleware[1].(types.Middleware[types.SlackCustomFunctionMiddlewareArgs]); ok {
			handler = h
		}
	}

	if handler == nil {
		return a // Invalid parameters, skip
	}
	if options == nil {
		options = &types.CustomFunctionOptions{AutoAcknowledge: true}
	}

	// Create a listener for function_executed events with this callback ID
	a.mu.Lock()
	defer a.mu.Unlock()

	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeEvent,
		constraints: listenerConstraints{
			eventType:  stringPtr("function_executed"),
			callbackID: &callbackID,
		},
		middleware: make([]types.Middleware[types.AllMiddlewareArgs], 0),
	}

	// Add built-in middleware for function processing
	if options.AutoAcknowledge {
		listener.middleware = append(listener.middleware, a.createAutoAckMiddleware())
	}

	// Add the custom function handler
	listener.middleware = append(listener.middleware, a.wrapCustomFunctionMiddleware(handler))

	a.listenerEntries = append(a.listenerEntries, listener)

	return a
}

// createAutoAckMiddleware creates middleware that auto-acknowledges events
func (a *App) createAutoAckMiddleware() types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Auto-acknowledge the event
		if args.Context != nil && args.Context.Custom != nil {
			if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
				if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
					if eventArgs.Ack != nil {
						var response interface{}
						if err := eventArgs.Ack(&response); err != nil {
							// Log the error but continue processing
							a.Logger.Error("Failed to acknowledge event", "error", err)
						}
					}
				}
			}
		}
		return args.Next()
	}
}

// wrapCustomFunctionMiddleware wraps custom function middleware
func (a *App) wrapCustomFunctionMiddleware(m types.Middleware[types.SlackCustomFunctionMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				// Create custom function args from event args
				customFunctionArgs := types.SlackCustomFunctionMiddlewareArgs{
					AllMiddlewareArgs: args,
					Event:             eventArgs.Event,
					Body:              eventArgs.Body,
					Payload:           eventArgs.Event, // Function payload is in the event
					Ack:               eventArgs.Ack,
					Complete: func(outputs map[string]interface{}) error {
						// TODO: Call Slack API to complete the function
						return nil
					},
					Fail: func(error string) error {
						// TODO: Call Slack API to fail the function
						return nil
					},
				}

				return m(customFunctionArgs)
			}
		}

		// Fallback: create basic custom function args
		customFunctionArgs := types.SlackCustomFunctionMiddlewareArgs{
			AllMiddlewareArgs: args,
			Complete: func(outputs map[string]interface{}) error {
				// TODO: Call Slack API to complete the function
				return nil
			},
			Fail: func(error string) error {
				// TODO: Call Slack API to fail the function
				return nil
			},
		}
		return m(customFunctionArgs)
	}
}

// Start starts the app
func (a *App) Start(ctx context.Context) error {
	if !a.initialized {
		if err := a.Init(ctx); err != nil {
			return err
		}
	}

	return a.receiver.Start(ctx)
}

// Stop stops the app
func (a *App) Stop(ctx context.Context) error {
	return a.receiver.Stop(ctx)
}

// ProcessEvent processes an incoming event - this is the core of the framework
func (a *App) ProcessEvent(ctx context.Context, event types.ReceiverEvent) error {
	if !a.initialized {
		return bolterrors.NewAppInitializationError("app not initialized")
	}

	if a.developerMode {
		a.Logger.Debug("Processing event", "body", string(event.Body))
	}

	// First check if the body can be parsed as JSON (for proper error handling)
	if len(event.Body) == 0 {
		// Empty body should return an error
		a.Logger.Warn("Empty request body. No listeners will be called.")
		return bolterrors.NewBaseError(bolterrors.EventProcessingError, "empty request body")
	}

	// Try to parse as JSON first to detect malformed JSON
	// But only if the content type suggests JSON
	contentType := ""
	if len(event.Headers) > 0 {
		for k, v := range event.Headers {
			if strings.EqualFold(k, "content-type") {
				contentType = v
				break
			}
		}
	}

	// Only validate JSON if content-type is application/json
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		var jsonTest map[string]interface{}
		if err := json.Unmarshal(event.Body, &jsonTest); err != nil {
			// If it's not valid JSON but claims to be JSON, this is malformed
			a.Logger.Warn("Malformed JSON in request body. No listeners will be called.")
			return bolterrors.NewBaseError(bolterrors.EventProcessingError, "malformed JSON in request body")
		}
	}

	// Determine event type and conversation context
	typeAndConv := helpers.GetTypeAndConversation(event.Body)
	if typeAndConv.Type == nil {
		// Body was parsed but event type is unknown - this is OK, just log and continue
		a.Logger.Warn("Could not determine the type of an incoming event. No listeners will be called.")
		return nil
	}

	// Check if this is an enterprise install
	isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(event.Body)

	// Build authorization source data
	source := a.buildAuthorizationSource(*typeAndConv.Type, typeAndConv.ConversationID, event.Body, isEnterpriseInstall)

	// Skip authorization for certain event types
	var authorizeResult *AuthorizeResult
	if *typeAndConv.Type == helpers.IncomingEventTypeEvent {
		eventType := helpers.ExtractEventType(event.Body)
		if helpers.IsEventTypeToSkipAuthorize(eventType) {
			// Use minimal authorization for events like app_uninstalled
			authorizeResult = &AuthorizeResult{
				TeamID:       source.TeamID,
				EnterpriseID: source.EnterpriseID,
			}
		} else {
			// Full authorization
			var err error
			authorizeResult, err = a.authorize(ctx, source, event.Body)
			if err != nil {
				return bolterrors.NewAuthorizationError("Failed to authorize", err)
			}
		}
	} else {
		// Full authorization for non-events
		var err error
		authorizeResult, err = a.authorize(ctx, source, event.Body)
		if err != nil {
			return bolterrors.NewAuthorizationError("Failed to authorize", err)
		}
	}

	// Create the context for this event
	appContext := a.buildEventContext(authorizeResult, event, *typeAndConv.Type)

	// Build the appropriate middleware arguments based on event type
	middlewareArgs, err := a.buildMiddlewareArgs(ctx, *typeAndConv.Type, event, appContext, authorizeResult)
	if err != nil {
		return err
	}

	// Process listeners - global middleware will be executed for each listener
	return a.processMatchingListeners(middlewareArgs, *typeAndConv.Type)
}

// Helper methods

func (a *App) initReceiver(options AppOptions) (types.Receiver, error) {
	if options.SocketMode {
		// Create Socket Mode receiver
		if options.AppToken == nil {
			return nil, bolterrors.NewAppInitializationError("app token required for socket mode")
		}

		receiverOptions := types.SocketModeReceiverOptions{
			AppToken:         *options.AppToken,
			Logger:           options.Logger,
			LogLevel:         options.LogLevel,
			CustomProperties: make(map[string]interface{}),
		}

		// Create the actual Socket Mode receiver
		return receivers.NewSocketModeReceiver(receiverOptions), nil
	} else {
		// Create HTTP receiver
		if options.SigningSecret == nil {
			return nil, bolterrors.NewAppInitializationError("signing secret required for HTTP receiver")
		}

		receiverOptions := types.HTTPReceiverOptions{
			SigningSecret:                 *options.SigningSecret,
			Endpoints:                     options.Endpoints,
			ProcessBeforeResponse:         options.ProcessBeforeResponse,
			UnhandledRequestHandler:       nil,
			UnhandledRequestTimeoutMillis: 3001,
			CustomProperties:              make(map[string]interface{}),
		}

		// Create the actual HTTP receiver
		return receivers.NewHTTPReceiver(receiverOptions), nil
	}
}

func (a *App) initAuthorize(token *string, authorize AuthorizeFunc, botID, botUserID *string) (AuthorizeFunc, error) {
	if authorize != nil {
		return authorize, nil
	}

	if token != nil {
		// Single workspace authorization
		return func(ctx context.Context, source AuthorizeSourceData, body interface{}) (*AuthorizeResult, error) {
			return &AuthorizeResult{
				BotToken:     token,
				BotID:        botID,
				BotUserID:    botUserID,
				TeamID:       source.TeamID,
				EnterpriseID: source.EnterpriseID,
				UserID:       source.UserID,
			}, nil
		}, nil
	}

	return nil, bolterrors.NewAppInitializationError("either token or authorize function must be provided")
}

func (a *App) defaultErrorHandler(err error) error {
	a.Logger.Error("Unhandled error", "error", err)
	return nil
}

// Wrapper methods to convert specific middleware to AllMiddlewareArgs

func (a *App) wrapEventMiddleware(m types.Middleware[types.SlackEventMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if eventArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackEventMiddlewareArgs); ok {
			// Update the base args
			eventArgs.AllMiddlewareArgs = args
			return m(eventArgs)
		}

		// Fallback: create basic event args
		eventArgs := types.SlackEventMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(eventArgs)
	}
}

func (a *App) wrapActionMiddleware(m types.Middleware[types.SlackActionMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if actionArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackActionMiddlewareArgs); ok {
			// Update the base args
			actionArgs.AllMiddlewareArgs = args
			return m(actionArgs)
		}

		// Fallback: create basic action args
		actionArgs := types.SlackActionMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(actionArgs)
	}
}

func (a *App) wrapCommandMiddleware(m types.Middleware[types.SlackCommandMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if commandArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackCommandMiddlewareArgs); ok {
			// Update the base args
			commandArgs.AllMiddlewareArgs = args
			return m(commandArgs)
		}

		// Fallback: create basic command args
		commandArgs := types.SlackCommandMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(commandArgs)
	}
}

func (a *App) wrapShortcutMiddleware(m types.Middleware[types.SlackShortcutMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if shortcutArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackShortcutMiddlewareArgs); ok {
			// Update the base args
			shortcutArgs.AllMiddlewareArgs = args
			return m(shortcutArgs)
		}

		// Fallback: create basic shortcut args
		shortcutArgs := types.SlackShortcutMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(shortcutArgs)
	}
}

func (a *App) wrapViewMiddleware(m types.Middleware[types.SlackViewMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if viewArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackViewMiddlewareArgs); ok {
			// Update the base args
			viewArgs.AllMiddlewareArgs = args
			return m(viewArgs)
		}

		// Fallback: create basic view args
		viewArgs := types.SlackViewMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(viewArgs)
	}
}

func (a *App) wrapOptionsMiddleware(m types.Middleware[types.SlackOptionsMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// The middleware args should be stored in the context
		if optionsArgs, ok := args.Context.Custom["middlewareArgs"].(types.SlackOptionsMiddlewareArgs); ok {
			// Update the base args
			optionsArgs.AllMiddlewareArgs = args
			return m(optionsArgs)
		}

		// Fallback: create basic options args
		optionsArgs := types.SlackOptionsMiddlewareArgs{
			AllMiddlewareArgs: args,
		}
		return m(optionsArgs)
	}
}

// Core processing methods

func (a *App) getClientForContext(context *types.Context) *slack.Client {
	// Return appropriate client based on context
	if context.BotToken != nil {
		return a.getOrCreateClient(*context.BotToken)
	}
	return a.Client
}

func (a *App) getOrCreateClient(token string) *slack.Client {
	// Use the team ID or enterprise ID as the pool key
	poolKey := "default"
	if pool, exists := a.clients[poolKey]; exists {
		return pool.GetOrCreate(token, a.clientOptions...)
	}

	// Create new pool
	pool := NewWebClientPool()
	a.clients[poolKey] = pool
	return pool.GetOrCreate(token, a.clientOptions...)
}

// buildAuthorizationSource builds the authorization source data
func (a *App) buildAuthorizationSource(eventType helpers.IncomingEventType, conversationID *string, body []byte, isEnterpriseInstall bool) AuthorizeSourceData {
	// Parse body as JSON or form data
	parsed := helpers.ParseRequestBody(body)

	source := AuthorizeSourceData{
		IsEnterpriseInstall: isEnterpriseInstall,
		ConversationID:      conversationID,
	}

	// Extract team_id based on event type
	switch eventType {
	case helpers.IncomingEventTypeEvent:
		if teamID := helpers.ExtractTeamID(body); teamID != nil {
			source.TeamID = teamID
		}
		if enterpriseID := helpers.ExtractEnterpriseID(body); enterpriseID != nil {
			source.EnterpriseID = enterpriseID
		}
		if userID := helpers.ExtractUserID(body); userID != nil {
			source.UserID = userID
		}
	case helpers.IncomingEventTypeCommand:
		if teamID, exists := parsed["team_id"]; exists {
			if teamIDStr, ok := teamID.(string); ok {
				source.TeamID = &teamIDStr
			}
		}
		if userID, exists := parsed["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				source.UserID = &userIDStr
			}
		}
	default:
		// For actions, shortcuts, views, options - extract from user/team objects
		if team, exists := parsed["team"]; exists {
			if teamMap, ok := team.(map[string]interface{}); ok {
				if id, exists := teamMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						source.TeamID = &idStr
					}
				}
			}
		}
		if user, exists := parsed["user"]; exists {
			if userMap, ok := user.(map[string]interface{}); ok {
				if id, exists := userMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						source.UserID = &idStr
					}
				}
				if teamID, exists := userMap["team_id"]; exists {
					if teamIDStr, ok := teamID.(string); ok && source.TeamID == nil {
						source.TeamID = &teamIDStr
					}
				}
			}
		}
	}

	return source
}

// buildEventContext creates the context for an event
func (a *App) buildEventContext(authResult *AuthorizeResult, event types.ReceiverEvent, eventType helpers.IncomingEventType) *types.Context {
	context := &types.Context{
		Custom: make(types.StringIndexed),
	}

	// Store the event type and body in context for middleware access
	context.Custom["eventType"] = eventType
	context.Custom["body"] = event.Body

	if authResult != nil {
		context.BotToken = authResult.BotToken
		context.UserToken = authResult.UserToken
		context.BotID = authResult.BotID
		context.BotUserID = authResult.BotUserID
		context.UserID = authResult.UserID
		context.TeamID = authResult.TeamID
		context.EnterpriseID = authResult.EnterpriseID
		context.IsEnterpriseInstall = authResult.Custom != nil

		// Add custom properties from auth result
		if authResult.Custom != nil {
			for k, v := range authResult.Custom {
				context.Custom[k] = v
			}
		}
	}

	// Add retry information if present
	if event.RetryNum != nil {
		context.RetryNum = event.RetryNum
	}
	if event.RetryReason != nil {
		context.RetryReason = event.RetryReason
	}

	// Add the body to context for middleware access
	context.Custom["body"] = event.Body

	return context
}

// buildMiddlewareArgs builds the appropriate middleware arguments based on event type
func (a *App) buildMiddlewareArgs(ctx context.Context, eventType helpers.IncomingEventType, event types.ReceiverEvent, appContext *types.Context, authResult *AuthorizeResult) (interface{}, error) {
	baseArgs := types.AllMiddlewareArgs{
		Context: appContext,
		Logger:  a.Logger,
		Client:  a.getClientForContext(appContext),
		Next:    func() error { return nil }, // Will be overridden in middleware chain
	}

	// Parse body as JSON or form data
	parsed := helpers.ParseRequestBody(event.Body)

	// Extract channel information early for Say function
	if eventType == helpers.IncomingEventTypeEvent {
		if eventData, exists := parsed["event"]; exists {
			if eventMap, ok := eventData.(map[string]interface{}); ok {
				if channel, exists := eventMap["channel"]; exists {
					if channelStr, ok := channel.(string); ok {
						appContext.Custom["channel"] = channelStr
					}
				}
			}
		}
	}

	// Create say function if there's a conversation context
	var sayFn types.SayFn
	if appContext.BotToken != nil {
		client := a.getClientForContext(appContext)
		sayFn = a.createSayFunction(client, appContext)
	}

	// Create respond function if there's a response URL
	var respondFn types.RespondFn
	if responseURL := a.extractResponseURL(parsed); responseURL != "" {
		respondFn = a.createRespondFunction(responseURL)
	}

	switch eventType {
	case helpers.IncomingEventTypeEvent:
		eventData := parsed["event"]
		args := types.SlackEventMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Event:             eventData,
			Body:              parsed,
			Say:               sayFn,
			Ack:               a.createEventAckFunction(event.Ack),
		}

		// Check if this is a message event and populate Message field
		if eventMap, ok := eventData.(map[string]interface{}); ok {
			if eventType, exists := eventMap["type"]; exists {
				if typeStr, ok := eventType.(string); ok && typeStr == "message" {
					messageEvent := &types.MessageEvent{}
					// Convert event data to message event
					if eventBytes, err := json.Marshal(eventData); err == nil {
						if err := json.Unmarshal(eventBytes, messageEvent); err == nil {
							args.Message = messageEvent
						}
					}
				}
			}
		}

		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = args
		return args, nil
	case helpers.IncomingEventTypeAction:
		var actionData interface{}
		if actions, exists := parsed["actions"]; exists {
			if actionList, ok := actions.([]interface{}); ok && len(actionList) > 0 {
				actionData = actionList[0]
			}
		}
		actionArgs := types.SlackActionMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Action:            actionData,
			Body:              parsed,
			Respond:           respondFn,
			Ack:               a.createActionAckFunction(event.Ack),
			Say:               &sayFn,
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = actionArgs
		return actionArgs, nil
	case helpers.IncomingEventTypeCommand:
		command := types.SlashCommand{}
		if cmd, ok := parsed["command"].(string); ok {
			command.Command = cmd
		}
		if text, ok := parsed["text"].(string); ok {
			command.Text = text
		}
		if userID, ok := parsed["user_id"].(string); ok {
			command.UserID = userID
		}
		if channelID, ok := parsed["channel_id"].(string); ok {
			command.ChannelID = channelID
		}
		if teamID, ok := parsed["team_id"].(string); ok {
			command.TeamID = teamID
		}
		if responseURL, ok := parsed["response_url"].(string); ok {
			command.ResponseURL = responseURL
		}
		if triggerID, ok := parsed["trigger_id"].(string); ok {
			command.TriggerID = triggerID
		}
		commandArgs := types.SlackCommandMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Command:           command,
			Body:              parsed,
			Respond:           respondFn,
			Ack:               a.createCommandAckFunction(event.Ack),
			Say:               sayFn,
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = commandArgs
		return commandArgs, nil
	case helpers.IncomingEventTypeShortcut:
		shortcutArgs, err := a.buildShortcutMiddlewareArgs(baseArgs, event, parsed, sayFn)
		if err != nil {
			return &types.SayResponse{}, err
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = shortcutArgs
		return shortcutArgs, nil
	case helpers.IncomingEventTypeViewAction:
		viewArgs := types.SlackViewMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			View:              parsed["view"], // Simplified for now
			Body:              parsed,
			Ack:               a.createViewAckFunction(event.Ack),
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = viewArgs
		return viewArgs, nil
	case helpers.IncomingEventTypeOptions:
		options := types.OptionsRequest{}
		if actionID, ok := parsed["action_id"].(string); ok {
			options.ActionID = actionID
		}
		if blockID, ok := parsed["block_id"].(string); ok {
			options.BlockID = blockID
		}
		if value, ok := parsed["value"].(string); ok {
			options.Value = value
		}
		optionsArgs := types.SlackOptionsMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Options:           options,
			Body:              parsed,
			Ack:               a.createOptionsAckFunction(event.Ack),
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = optionsArgs
		return optionsArgs, nil
	default:
		return baseArgs, nil
	}
}

// processGlobalMiddleware processes global middleware
// Returns (shouldContinue, error) where shouldContinue indicates if listeners should be processed

// processMatchingListeners processes listeners that match the event
func (a *App) processMatchingListeners(middlewareArgs interface{}, eventType helpers.IncomingEventType) error {
	var matchingListeners []*listenerEntry

	// Find listeners that match this event type and constraints
	for _, listener := range a.listenerEntries {
		if a.listenerMatchesEvent(listener, middlewareArgs, eventType) {
			matchingListeners = append(matchingListeners, listener)
		}
	}

	// Also check legacy listeners for backward compatibility
	for _, listenerChain := range a.listeners {
		if a.listenerMatches(listenerChain, middlewareArgs, eventType) {
			// Convert to listenerEntry format for execution
			legacyListener := &listenerEntry{
				eventType:  eventType,
				middleware: listenerChain,
			}
			matchingListeners = append(matchingListeners, legacyListener)
		}
	}

	// If there are no matching listeners, still execute global middleware
	if len(matchingListeners) == 0 {
		// Create an empty listener to ensure global middleware runs
		emptyListener := &listenerEntry{
			eventType:  eventType,
			middleware: []types.Middleware[types.AllMiddlewareArgs]{}, // Empty listener middleware
		}
		matchingListeners = append(matchingListeners, emptyListener)
	}

	// Execute all matching listeners (including the empty one if no real listeners match)
	var listenerErrors []error
	for _, listener := range matchingListeners {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Convert panic to error
					listenerErrors = append(listenerErrors, fmt.Errorf("listener panic: %v", r))
				}
			}()
			if err := a.executeListenerChain(listener.middleware, middlewareArgs); err != nil {
				listenerErrors = append(listenerErrors, err)
			}
		}()
	}

	if len(listenerErrors) > 0 {
		return bolterrors.NewMultipleListenerError(listenerErrors)
	}

	return nil
}

// executeMiddlewareChain executes a middleware chain

// executeMiddlewareChainWithCompletion executes a middleware chain and tracks completion
// Returns (completed, error) where completed indicates if the entire chain was executed

// executeListenerChain executes a listener chain with proper argument conversion
// First executes global middleware, then the listener-specific middleware
func (a *App) executeListenerChain(chain []types.Middleware[types.AllMiddlewareArgs], middlewareArgs interface{}) error {
	// Combine global middleware with listener middleware
	fullChain := make([]types.Middleware[types.AllMiddlewareArgs], 0, len(a.middleware)+len(chain))
	fullChain = append(fullChain, a.middleware...)
	fullChain = append(fullChain, chain...)

	index := 0

	var next types.NextFn
	next = func() error {
		if index >= len(fullChain) {
			return nil
		}

		currentMiddleware := fullChain[index]
		index++

		// Convert middleware args to base args for execution
		baseArgs := a.extractBaseArgs(middlewareArgs)
		baseArgs.Next = next

		return currentMiddleware(baseArgs)
	}

	return next()
}

// Helper methods for building specific middleware args
func (a *App) buildShortcutMiddlewareArgs(baseArgs types.AllMiddlewareArgs, event types.ReceiverEvent, parsed map[string]interface{}, sayFn types.SayFn) (types.SlackShortcutMiddlewareArgs, error) {
	args := types.SlackShortcutMiddlewareArgs{
		AllMiddlewareArgs: baseArgs,
		Body:              parsed,
		Payload:           parsed,
		Ack:               a.createAckFunction(event),
	}

	// Add say function for message shortcuts
	if shortcutType, exists := parsed["type"]; exists {
		if typeStr, ok := shortcutType.(string); ok && typeStr == "message_action" {
			args.Say = &sayFn
		}
	}

	// Build shortcut object
	if shortcutType, exists := parsed["type"]; exists {
		if typeStr, ok := shortcutType.(string); ok {
			if typeStr == "shortcut" {
				var globalShortcut types.GlobalShortcut
				if shortcutBytes, err := json.Marshal(parsed); err == nil {
					if err := json.Unmarshal(shortcutBytes, &globalShortcut); err == nil {
						args.Shortcut = globalShortcut
					}
				}
			} else if typeStr == "message_action" {
				var messageShortcut types.MessageShortcut
				if shortcutBytes, err := json.Marshal(parsed); err == nil {
					if err := json.Unmarshal(shortcutBytes, &messageShortcut); err == nil {
						args.Shortcut = messageShortcut
					}
				}
			}
		}
	}

	return args, nil
}

// Utility functions
func (a *App) extractResponseURL(parsed map[string]interface{}) string {
	if responseURL, exists := parsed["response_url"]; exists {
		if responseURLStr, ok := responseURL.(string); ok {
			return responseURLStr
		}
	}
	return ""
}

func (a *App) extractBaseArgs(middlewareArgs interface{}) types.AllMiddlewareArgs {
	switch args := middlewareArgs.(type) {
	case types.SlackEventMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.SlackActionMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.SlackCommandMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.SlackShortcutMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.SlackViewMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.SlackOptionsMiddlewareArgs:
		return args.AllMiddlewareArgs
	case types.AllMiddlewareArgs:
		return args
	default:
		return types.AllMiddlewareArgs{}
	}
}

// createSayFunction creates a say function for sending messages
func (a *App) createSayFunction(client *slack.Client, context *types.Context) types.SayFn {
	return func(message types.SayMessage) (*types.SayResponse, error) {
		// Determine channel from context or message
		var channelID string

		// Try to get channel from message
		switch msg := message.(type) {
		case types.SayString:
			// Simple text message - need channel from context
			if context.Custom != nil {
				if ch, exists := context.Custom["channel"]; exists {
					if chStr, ok := ch.(string); ok {
						channelID = chStr
					}
				}
			}
			if channelID == "" {
				return &types.SayResponse{}, bolterrors.NewAppInitializationError("no channel context for say function")
			}

			_, _, err := client.PostMessage(channelID, slack.MsgOptionText(string(msg), false))
			return &types.SayResponse{}, err

		case types.SayArguments:
			if msg.Channel != nil {
				channelID = *msg.Channel
			}

			var options []slack.MsgOption
			if msg.Text != nil {
				options = append(options, slack.MsgOptionText(*msg.Text, false))
			}
			if len(msg.Blocks) > 0 {
				options = append(options, slack.MsgOptionBlocks(msg.Blocks...))
			}
			if len(msg.Attachments) > 0 {
				options = append(options, slack.MsgOptionAttachments(msg.Attachments...))
			}
			if msg.ThreadTS != nil {
				options = append(options, slack.MsgOptionTS(*msg.ThreadTS))
			}
			if msg.Metadata != nil {
				options = append(options, slack.MsgOptionMetadata(*msg.Metadata))
			}

			_, _, err := client.PostMessage(channelID, options...)
			return &types.SayResponse{}, err

		}

		return &types.SayResponse{}, bolterrors.NewAppInitializationError("unsupported message type for say function")
	}
}

// createRespondFunction creates a respond function for response URLs
func (a *App) createRespondFunction(responseURL string) types.RespondFn {
	return func(message types.RespondMessage) error {
		var payload []byte
		var err error

		switch msg := message.(type) {
		case types.RespondString:
			payload, err = json.Marshal(map[string]interface{}{
				"text": string(msg),
			})
		case types.RespondArguments:
			payload, err = json.Marshal(msg)
		default:
			payload, err = json.Marshal(message)
		}

		if err != nil {
			return err
		}

		// Validate URL to prevent potential security issues
		if !strings.HasPrefix(responseURL, "https://hooks.slack.com/") {
			return bolterrors.NewAppInitializationError("invalid response URL")
		}

		// Use a client with timeout for security
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// Create context-aware request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return bolterrors.NewAppInitializationError("failed to send response")
		}

		return nil
	}
}

// createAckFunction creates a generic ack function
func (a *App) createAckFunction(event types.ReceiverEvent) types.AckFn[interface{}] {
	return func(response *interface{}) error {
		if response != nil {
			return event.Ack(*response)
		}
		return event.Ack(nil)
	}
}

// createCommandAckFunction creates an ack function for commands
func (a *App) createCommandAckFunction(receiverAck func(response interface{}) error) types.AckFn[types.CommandResponse] {
	return func(response *types.CommandResponse) error {
		return receiverAck(response)
	}
}

// createViewAckFunction creates an ack function for views
func (a *App) createViewAckFunction(receiverAck func(response interface{}) error) types.AckFn[types.ViewResponse] {
	return func(response *types.ViewResponse) error {
		return receiverAck(response)
	}
}

// createOptionsAckFunction creates an ack function for options
func (a *App) createOptionsAckFunction(receiverAck func(response interface{}) error) types.AckFn[types.OptionsResponse] {
	return func(response *types.OptionsResponse) error {
		return receiverAck(response)
	}
}

// createEventAckFunction creates an ack function for events
func (a *App) createEventAckFunction(receiverAck func(response interface{}) error) types.AckFn[interface{}] {
	return func(response *interface{}) error {
		if response != nil {
			return receiverAck(*response)
		}
		return receiverAck(nil)
	}
}

// createActionAckFunction creates an ack function for actions
func (a *App) createActionAckFunction(receiverAck func(response interface{}) error) types.AckFn[interface{}] {
	return func(response *interface{}) error {
		if response != nil {
			return receiverAck(*response)
		}
		return receiverAck(nil)
	}
}

// listenerMatchesEvent checks if a listener entry matches the current event
func (a *App) listenerMatchesEvent(listener *listenerEntry, middlewareArgs interface{}, eventType helpers.IncomingEventType) bool {

	// First check if the event types match
	if listener.eventType != eventType {
		return false
	}

	switch eventType {
	case helpers.IncomingEventTypeEvent:
		return a.matchesEventConstraints(listener, middlewareArgs)
	case helpers.IncomingEventTypeAction:
		return a.matchesActionConstraints(listener, middlewareArgs)
	case helpers.IncomingEventTypeCommand:
		return a.matchesCommandConstraints(listener, middlewareArgs)
	case helpers.IncomingEventTypeShortcut:
		return a.matchesShortcutConstraints(listener, middlewareArgs)
	case helpers.IncomingEventTypeViewAction:
		return a.matchesViewConstraints(listener, middlewareArgs)
	case helpers.IncomingEventTypeOptions:
		return a.matchesOptionsConstraints(listener, middlewareArgs)
	default:
		return false
	}
}

// matchesEventConstraints checks if an event matches the listener's event constraints
func (a *App) matchesEventConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs)
	if !ok {
		return false
	}

	// Extract event type from the event data
	var eventTypeStr string
	if eventMap, ok := eventArgs.Event.(map[string]interface{}); ok {
		if eventType, exists := eventMap["type"]; exists {
			if typeStr, ok := eventType.(string); ok {
				eventTypeStr = typeStr
			}
		}
	}

	// Check event type constraint (string)
	if listener.constraints.eventType != nil {
		if eventTypeStr != *listener.constraints.eventType {
			return false
		}
	}

	// Check event type pattern (RegExp)
	if listener.constraints.eventTypePattern != nil {
		if eventTypeStr == "" || !listener.constraints.eventTypePattern.MatchString(eventTypeStr) {
			return false
		}
	}

	// Check message pattern constraint for message events
	if listener.constraints.messagePattern != nil {
		if eventArgs.Message == nil {
			return false
		}

		return helpers.MatchesPattern(eventArgs.Message.Text, listener.constraints.messagePattern)
	}

	// Check callback ID constraint for function_executed events
	if listener.constraints.callbackID != nil && eventTypeStr == "function_executed" {
		if eventMap, ok := eventArgs.Event.(map[string]interface{}); ok {
			if function, exists := eventMap["function"]; exists {
				if functionMap, ok := function.(map[string]interface{}); ok {
					if callbackID, exists := functionMap["callback_id"]; exists {
						if callbackIDStr, ok := callbackID.(string); ok {
							return callbackIDStr == *listener.constraints.callbackID
						}
					}
				}
			}
		}
		return false
	}

	return true
}

// matchesActionConstraints checks if an action matches the listener's action constraints
func (a *App) matchesActionConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	actionArgs, ok := middlewareArgs.(types.SlackActionMiddlewareArgs)
	if !ok {
		return false
	}

	// Check action type constraint first (e.g., "block_actions")
	if listener.constraints.actionType != nil {
		bodyMap, ok := actionArgs.Body.(map[string]interface{})
		if !ok {
			return false
		}

		actionType, exists := bodyMap["type"]
		if !exists {
			return false
		}

		actionTypeStr, ok := actionType.(string)
		if !ok || actionTypeStr != *listener.constraints.actionType {
			return false
		}
	}

	// If there are no specific field constraints, match on type only
	if listener.constraints.actionID == nil && listener.constraints.blockID == nil && listener.constraints.callbackID == nil &&
		listener.constraints.actionIDPattern == nil && listener.constraints.blockIDPattern == nil && listener.constraints.callbackIDPattern == nil {
		return true
	}

	actionMap, ok := actionArgs.Action.(map[string]interface{})
	if !ok {
		return false
	}

	// Check action_id constraint (string or regexp)
	if listener.constraints.actionID != nil {
		actionID, exists := actionMap["action_id"]
		if !exists {
			return false
		}
		actionIDStr, ok := actionID.(string)
		if !ok {
			return false
		}
		if actionIDStr != *listener.constraints.actionID {
			return false
		}
	} else if listener.constraints.actionIDPattern != nil {
		actionID, exists := actionMap["action_id"]
		if !exists {
			return false
		}
		actionIDStr, ok := actionID.(string)
		if !ok {
			return false
		}
		if !listener.constraints.actionIDPattern.MatchString(actionIDStr) {
			return false
		}
	}

	// Check block_id constraint (string or regexp)
	if listener.constraints.blockID != nil {
		blockID, exists := actionMap["block_id"]
		if !exists {
			return false
		}
		blockIDStr, ok := blockID.(string)
		if !ok {
			return false
		}
		if blockIDStr != *listener.constraints.blockID {
			return false
		}
	} else if listener.constraints.blockIDPattern != nil {
		blockID, exists := actionMap["block_id"]
		if !exists {
			return false
		}
		blockIDStr, ok := blockID.(string)
		if !ok {
			return false
		}
		if !listener.constraints.blockIDPattern.MatchString(blockIDStr) {
			return false
		}
	}

	// Check callback_id constraint (string or regexp) for legacy actions
	if listener.constraints.callbackID != nil {
		// Check in payload first
		if bodyMap, ok := actionArgs.Body.(map[string]interface{}); ok {
			if callbackID, exists := bodyMap["callback_id"]; exists {
				callbackIDStr, ok := callbackID.(string)
				if ok && callbackIDStr == *listener.constraints.callbackID {
					return true
				}
			}
		}
		return false
	} else if listener.constraints.callbackIDPattern != nil {
		// Check in payload first
		if bodyMap, ok := actionArgs.Body.(map[string]interface{}); ok {
			if callbackID, exists := bodyMap["callback_id"]; exists {
				callbackIDStr, ok := callbackID.(string)
				if ok && listener.constraints.callbackIDPattern.MatchString(callbackIDStr) {
					return true
				}
			}
		}
		return false
	}

	return true
}

// matchesCommandConstraints checks if a command matches the listener's command constraints
func (a *App) matchesCommandConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	commandArgs, ok := middlewareArgs.(types.SlackCommandMiddlewareArgs)
	if !ok {
		return false
	}

	// Check command constraint (string)
	if listener.constraints.command != nil {
		if commandArgs.Command.Command != *listener.constraints.command {
			return false
		}
	}

	// Check command pattern (RegExp)
	if listener.constraints.commandPattern != nil {
		if !listener.constraints.commandPattern.MatchString(commandArgs.Command.Command) {
			return false
		}
	}

	return true
}

// matchesShortcutConstraints checks if a shortcut matches the listener's shortcut constraints
func (a *App) matchesShortcutConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	shortcutArgs, ok := middlewareArgs.(types.SlackShortcutMiddlewareArgs)
	if !ok {
		return false
	}

	bodyMap, ok := shortcutArgs.Body.(map[string]interface{})
	if !ok {
		return false
	}

	// Extract callback_id from body
	var callbackIDStr string
	if callbackID, exists := bodyMap["callback_id"]; exists {
		if idStr, ok := callbackID.(string); ok {
			callbackIDStr = idStr
		}
	}

	// Check callback_id constraint (string)
	if listener.constraints.callbackID != nil {
		if callbackIDStr != *listener.constraints.callbackID {
			return false
		}
	}

	// Check callback_id pattern (RegExp)
	if listener.constraints.callbackIDPattern != nil {
		if callbackIDStr == "" || !listener.constraints.callbackIDPattern.MatchString(callbackIDStr) {
			return false
		}
	}

	// Check shortcut type constraint
	if listener.constraints.shortcutType != nil {
		shortcutType, exists := bodyMap["type"]
		if !exists {
			return false
		}
		shortcutTypeStr, ok := shortcutType.(string)
		if !ok || shortcutTypeStr != *listener.constraints.shortcutType {
			return false
		}
	}

	return true
}

// matchesViewConstraints checks if a view matches the listener's view constraints
func (a *App) matchesViewConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	viewArgs, ok := middlewareArgs.(types.SlackViewMiddlewareArgs)
	if !ok {
		return false
	}

	bodyMap, ok := viewArgs.Body.(map[string]interface{})
	if !ok {
		return false
	}

	// Check view type constraint
	if listener.constraints.viewType != nil {
		viewType, exists := bodyMap["type"]
		if !exists {
			return false
		}
		viewTypeStr, ok := viewType.(string)
		if !ok || viewTypeStr != *listener.constraints.viewType {
			return false
		}
	}

	// Extract callback_id from view
	var callbackIDStr string
	if view, exists := bodyMap["view"]; exists {
		if viewMap, ok := view.(map[string]interface{}); ok {
			if callbackID, exists := viewMap["callback_id"]; exists {
				if idStr, ok := callbackID.(string); ok {
					callbackIDStr = idStr
				}
			}
		}
	}

	// Check callback_id constraint (string)
	if listener.constraints.callbackID != nil {
		if callbackIDStr != *listener.constraints.callbackID {
			return false
		}
	}

	// Check callback_id pattern (RegExp)
	if listener.constraints.callbackIDPattern != nil {
		if callbackIDStr == "" || !listener.constraints.callbackIDPattern.MatchString(callbackIDStr) {
			return false
		}
	}

	return true
}

// matchesOptionsConstraints checks if an options request matches the listener's options constraints
func (a *App) matchesOptionsConstraints(listener *listenerEntry, middlewareArgs interface{}) bool {
	optionsArgs, ok := middlewareArgs.(types.SlackOptionsMiddlewareArgs)
	if !ok {
		return false
	}

	bodyMap, ok := optionsArgs.Body.(map[string]interface{})
	if !ok {
		return false
	}

	// Extract action_id from body
	var actionIDStr string
	if actionID, exists := bodyMap["action_id"]; exists {
		if idStr, ok := actionID.(string); ok {
			actionIDStr = idStr
		}
	}

	// Check action_id constraint (string)
	if listener.constraints.actionID != nil {
		if actionIDStr != *listener.constraints.actionID {
			return false
		}
	}

	// Check action_id pattern (RegExp)
	if listener.constraints.actionIDPattern != nil {
		if actionIDStr == "" || !listener.constraints.actionIDPattern.MatchString(actionIDStr) {
			return false
		}
	}

	// Check block_id constraint
	if listener.constraints.blockID != nil {
		blockID, exists := bodyMap["block_id"]
		if !exists {
			return false
		}
		blockIDStr, ok := blockID.(string)
		if !ok || blockIDStr != *listener.constraints.blockID {
			return false
		}
	}

	return true
}

// listenerMatches checks if a listener chain matches the current event (legacy method)
func (a *App) listenerMatches(listenerChain []types.Middleware[types.AllMiddlewareArgs], middlewareArgs interface{}, eventType helpers.IncomingEventType) bool {
	// Legacy method - for backward compatibility, assume all match
	return true
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
