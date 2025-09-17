package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
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

// AppOptions represents configuration options for the App
type AppOptions struct {
	// Receiver configuration
	SigningSecret         string                   `json:"signing_secret,omitempty"`
	Endpoints             *types.ReceiverEndpoints `json:"endpoints,omitempty"`
	Port                  int                      `json:"port,omitempty"`
	CustomRoutes          []types.CustomRoute      `json:"custom_routes,omitempty"`
	ProcessBeforeResponse bool                     `json:"process_before_response"`
	SignatureVerification bool                     `json:"signature_verification"`

	// OAuth configuration
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	StateSecret  string   `json:"state_secret,omitempty"`
	RedirectURI  string   `json:"redirect_uri,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`

	// Client configuration
	HTTPClient    *http.Client   `json:"-"`
	ClientOptions []slack.Option `json:"-"`
	Token         string         `json:"token,omitempty"`
	AppToken      string         `json:"app_token,omitempty"`
	BotID         string         `json:"bot_id,omitempty"`
	BotUserID     string         `json:"bot_user_id,omitempty"`

	// Authorization
	Authorize AuthorizeFunc `json:"-"`

	// Receiver
	Receiver types.Receiver `json:"-"`

	// Logging
	Logger   *slog.Logger    `json:"-"`
	LogLevel *types.LogLevel `json:"log_level,omitempty"`

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
	TeamID              string `json:"team_id,omitempty"`
	EnterpriseID        string `json:"enterprise_id,omitempty"`
	UserID              string `json:"user_id,omitempty"`
	ConversationID      string `json:"conversation_id,omitempty"`
	IsEnterpriseInstall bool   `json:"is_enterprise_install"`
}

// AuthorizeResult represents the result of authorization
type AuthorizeResult struct {
	BotToken     string                 `json:"bot_token,omitempty"`
	UserToken    string                 `json:"user_token,omitempty"`
	BotID        string                 `json:"bot_id,omitempty"`
	BotUserID    string                 `json:"bot_user_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	TeamID       string                 `json:"team_id,omitempty"`
	EnterpriseID string                 `json:"enterprise_id,omitempty"`
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
	eventType      string
	messagePattern interface{}
	actionID       string
	blockID        string
	callbackID     string
	command        string
	shortcutType   string
	viewType       string
	actionType     string // For action type constraints (e.g., "block_actions")
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
	logLevel                 types.LogLevel
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
	if options.Token != "" && options.Authorize != nil {
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
		app.logLevel = types.LogLevelDebug
	} else {
		app.logLevel = types.LogLevelInfo
	}

	// Configure the logger level if using the default logger
	if options.Logger == nil {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: app.logLevel.ToSlogLevel(),
		})
		app.Logger = slog.New(handler)
	}

	// Set up client options
	app.clientOptions = []slack.Option{}
	if options.ClientOptions != nil {
		app.clientOptions = append(app.clientOptions, options.ClientOptions...)
	}

	// Create the main client
	if options.Token != "" {
		app.Client = slack.New(options.Token, app.clientOptions...)
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
		if options.Token != "" {
			app.argToken = &options.Token
		}
		app.argAuthorize = options.Authorize
		if options.Token != "" {
			app.argAuthorization = &AuthorizeResult{
				BotID:     options.BotID,
				BotUserID: options.BotUserID,
				BotToken:  options.Token,
			}
		}
		app.initialized = false
	} else {
		var token *string
		if options.Token != "" {
			token = &options.Token
		}
		var botID *string
		if options.BotID != "" {
			botID = &options.BotID
		}
		var botUserID *string
		if options.BotUserID != "" {
			botUserID = &options.BotUserID
		}
		authorize, err := app.initAuthorize(token, options.Authorize, botID, botUserID)
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
func (a *App) Event(eventType types.SlackEventType, middleware ...types.Middleware[types.SlackEventMiddlewareArgs]) *App {
	a.mu.Lock()
	defer a.mu.Unlock()

	eventTypeStr := eventType.String()

	// Create a listener entry with routing information
	listener := &listenerEntry{
		eventType: helpers.IncomingEventTypeEvent,
		constraints: listenerConstraints{
			eventType: eventTypeStr,
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
			eventType:      "message",
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
			command: command,
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
		CallbackID: callbackID,
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
		CallbackID: callbackID,
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
		ActionID: actionID,
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
			eventType:  "function_executed",
			callbackID: callbackID,
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
		if options.AppToken == "" {
			return nil, bolterrors.NewAppInitializationError("app token required for socket mode")
		}

		receiverOptions := types.SocketModeReceiverOptions{
			AppToken:         options.AppToken,
			BotToken:         options.Token,
			Logger:           options.Logger,
			LogLevel:         &[]types.LogLevel{types.LogLevelInfo}[0], // Default value
			CustomProperties: make(map[string]interface{}),
		}
		if options.LogLevel != nil {
			receiverOptions.LogLevel = options.LogLevel
		}

		// Create the actual Socket Mode receiver
		return receivers.NewSocketModeReceiver(receiverOptions), nil
	} else {
		// Create HTTP receiver
		if options.SigningSecret == "" {
			return nil, bolterrors.NewAppInitializationError("signing secret required for HTTP receiver")
		}

		receiverOptions := types.HTTPReceiverOptions{
			SigningSecret:                 options.SigningSecret,
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
				BotToken:     getStringValue(token),
				BotID:        getStringValue(botID),
				BotUserID:    getStringValue(botUserID),
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
	if context.BotToken != "" {
		return a.getOrCreateClient(context.BotToken)
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
		ConversationID:      getStringValue(conversationID),
	}

	// Extract team_id based on event type
	switch eventType {
	case helpers.IncomingEventTypeEvent:
		if teamID := helpers.ExtractTeamID(body); teamID != nil {
			source.TeamID = *teamID
		}
		if enterpriseID := helpers.ExtractEnterpriseID(body); enterpriseID != nil {
			source.EnterpriseID = *enterpriseID
		}
		if userID := helpers.ExtractUserID(body); userID != nil {
			source.UserID = *userID
		}
	case helpers.IncomingEventTypeCommand:
		if teamID, exists := parsed["team_id"]; exists {
			if teamIDStr, ok := teamID.(string); ok {
				source.TeamID = teamIDStr
			}
		}
		if userID, exists := parsed["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				source.UserID = userIDStr
			}
		}
	default:
		// For actions, shortcuts, views, options - extract from user/team objects
		if team, exists := parsed["team"]; exists {
			if teamMap, ok := team.(map[string]interface{}); ok {
				if id, exists := teamMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						source.TeamID = idStr
					}
				}
			}
		}
		if user, exists := parsed["user"]; exists {
			if userMap, ok := user.(map[string]interface{}); ok {
				if id, exists := userMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						source.UserID = idStr
					}
				}
				if teamID, exists := userMap["team_id"]; exists {
					if teamIDStr, ok := teamID.(string); ok && source.TeamID == "" {
						source.TeamID = teamIDStr
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
			maps.Copy(context.Custom, authResult.Custom)
		}
	}

	// Add retry information if present
	if event.RetryNum != 0 {
		context.RetryNum = event.RetryNum
	}
	if event.RetryReason != "" {
		context.RetryReason = event.RetryReason
	}

	// Extract function execution ID from body if present
	parsed := helpers.ParseRequestBody(event.Body)
	if functionExecutionID, exists := parsed["function_execution_id"]; exists {
		if functionExecutionIDStr, ok := functionExecutionID.(string); ok {
			context.FunctionExecutionID = functionExecutionIDStr
		}
	}

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
	} else if eventType == helpers.IncomingEventTypeCommand {
		// Extract channel information from slash command
		if channelID, exists := parsed["channel_id"]; exists {
			if channelStr, ok := channelID.(string); ok {
				appContext.Custom["channel"] = channelStr
			}
		}
	}

	// Create say function if there's a conversation context
	var sayFn types.SayFn
	if appContext.BotToken != "" {
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

		// Parse the inner event
		parsedEvent, err := helpers.ParseSlackEvent(eventData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse slack event: %w", err)
		}

		// Parse the event envelope
		eventEnvelope, err := helpers.ParseEventEnvelope(parsed)
		if err != nil {
			return nil, fmt.Errorf("failed to parse event envelope: %w", err)
		}

		args := types.SlackEventMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Event:             parsedEvent,   // Strongly typed event
			Body:              eventEnvelope, // Strongly typed event envelope
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

		// Parse the action data into strongly typed action
		action, err := helpers.ParseSlackAction(actionData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse slack action: %w", err)
		}

		// For the body, we need to parse the entire request as an action
		// This is a bit complex because the body structure varies by action type
		bodyAction, bodyErr := helpers.ParseSlackAction(parsed)
		if bodyErr != nil {
			// If we can't parse the full body as an action, use the individual action
			bodyAction = action
		}

		actionArgs := types.SlackActionMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Action:            action,     // Strongly typed action
			Payload:           action,     // Strongly typed payload
			Body:              bodyAction, // Strongly typed body
			Respond:           respondFn,
			Ack:               a.createActionAckFunction(event.Ack),
			Say:               sayFn,
		}
		// Store the full args in context for wrapper functions
		baseArgs.Context.Custom["middlewareArgs"] = actionArgs
		return actionArgs, nil
	case helpers.IncomingEventTypeCommand:
		command, err := helpers.ParseSlashCommand(parsed)
		if err != nil {
			return nil, fmt.Errorf("failed to parse slash command: %w", err)
		}

		commandArgs := types.SlackCommandMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			Command:           command,
			Body:              command, // Strongly typed body
			Payload:           command, // Strongly typed payload
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
		// Parse the view action (body)
		viewAction, err := helpers.ParseSlackView(parsed)
		if err != nil {
			return nil, fmt.Errorf("failed to parse view action: %w", err)
		}

		// Parse the view output from the view data
		viewOutput, err := helpers.ParseViewOutput(parsed["view"])
		if err != nil {
			return nil, fmt.Errorf("failed to parse view output: %w", err)
		}

		viewArgs := types.SlackViewMiddlewareArgs{
			AllMiddlewareArgs: baseArgs,
			View:              viewOutput, // Strongly typed processed view data
			Body:              viewAction, // Strongly typed view action
			Payload:           viewOutput, // Strongly typed payload (same as view)
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
	// Parse the shortcut using the helper
	shortcut, err := helpers.ParseSlackShortcut(parsed)
	if err != nil {
		return types.SlackShortcutMiddlewareArgs{}, fmt.Errorf("failed to parse shortcut: %w", err)
	}

	args := types.SlackShortcutMiddlewareArgs{
		AllMiddlewareArgs: baseArgs,
		Shortcut:          shortcut, // Strongly typed shortcut
		Body:              shortcut, // Strongly typed body
		Payload:           shortcut, // Strongly typed payload
		Ack:               a.createAckFunction(event),
	}

	// Add say function for message shortcuts
	if shortcutType, exists := parsed["type"]; exists {
		if typeStr, ok := shortcutType.(string); ok && typeStr == "message_action" {
			args.Say = &sayFn
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
			if msg.Channel != "" {
				channelID = msg.Channel
			}

			var options []slack.MsgOption
			if msg.Text != "" {
				options = append(options, slack.MsgOptionText(msg.Text, false))
			}
			if len(msg.Blocks) > 0 {
				options = append(options, slack.MsgOptionBlocks(msg.Blocks...))
			}
			if len(msg.Attachments) > 0 {
				options = append(options, slack.MsgOptionAttachments(msg.Attachments...))
			}
			if msg.ThreadTS != "" {
				options = append(options, slack.MsgOptionTS(msg.ThreadTS))
			}
			if msg.Metadata != nil {
				options = append(options, slack.MsgOptionMetadata(*msg.Metadata))
			}

			_, _, err := client.PostMessage(channelID, options...)
			return &types.SayResponse{}, err

		case *types.SayArguments:
			// Handle pointer to SayArguments
			if msg.Channel != "" {
				channelID = msg.Channel
			}

			var options []slack.MsgOption
			if msg.Text != "" {
				options = append(options, slack.MsgOptionText(msg.Text, false))
			}
			if len(msg.Blocks) > 0 {
				options = append(options, slack.MsgOptionBlocks(msg.Blocks...))
			}
			if len(msg.Attachments) > 0 {
				options = append(options, slack.MsgOptionAttachments(msg.Attachments...))
			}
			if msg.ThreadTS != "" {
				options = append(options, slack.MsgOptionTS(msg.ThreadTS))
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
		// Allow localhost/127.0.0.1 for testing, but require https://hooks.slack.com/ for production
		if !strings.HasPrefix(responseURL, "https://hooks.slack.com/") &&
			!strings.HasPrefix(responseURL, "http://127.0.0.1") &&
			!strings.HasPrefix(responseURL, "http://localhost") {
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
			ackResp := a.convertToAckResponse(*response)
			return event.Ack(ackResp)
		}
		return event.Ack(nil)
	}
}

// createCommandAckFunction creates an ack function for commands
func (a *App) createCommandAckFunction(receiverAck func(response types.AckResponse) error) types.AckFn[types.CommandResponse] {
	return func(response *types.CommandResponse) error {
		ackResp := a.convertToAckResponse(response)
		return receiverAck(ackResp)
	}
}

// createViewAckFunction creates an ack function for views
func (a *App) createViewAckFunction(receiverAck func(response types.AckResponse) error) types.AckFn[types.ViewResponse] {
	return func(response *types.ViewResponse) error {
		ackResp := a.convertToAckResponse(response)
		return receiverAck(ackResp)
	}
}

// createOptionsAckFunction creates an ack function for options
func (a *App) createOptionsAckFunction(receiverAck func(response types.AckResponse) error) types.AckFn[types.OptionsResponse] {
	return func(response *types.OptionsResponse) error {
		ackResp := a.convertToAckResponse(response)
		return receiverAck(ackResp)
	}
}

// convertToAckResponse converts an interface{} to AckResponse
func (a *App) convertToAckResponse(response interface{}) types.AckResponse {
	if response == nil {
		return types.AckVoid{}
	}

	switch resp := response.(type) {
	case string:
		return types.AckString(resp)
	case types.SayArguments:
		return resp // SayArguments implements AckResponse
	case types.RespondArguments:
		return resp // RespondArguments implements AckResponse
	case types.AckResponse:
		return resp
	default:
		// For other types that don't implement AckResponse, we need to handle them
		// This is a fallback that might need adjustment based on actual usage
		return types.AckString(fmt.Sprintf("%v", resp))
	}
}

// createEventAckFunction creates an ack function for events
func (a *App) createEventAckFunction(receiverAck func(response types.AckResponse) error) types.AckFn[interface{}] {
	return func(response *interface{}) error {
		if response != nil {
			// Convert interface{} to AckResponse
			ackResp := a.convertToAckResponse(*response)
			return receiverAck(ackResp)
		}
		return receiverAck(nil)
	}
}

// createActionAckFunction creates an ack function for actions
func (a *App) createActionAckFunction(receiverAck func(response types.AckResponse) error) types.AckFn[interface{}] {
	return func(response *interface{}) error {
		if response != nil {
			ackResp := a.convertToAckResponse(*response)
			return receiverAck(ackResp)
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
	var eventMap map[string]interface{}
	if genericEvent, ok := eventArgs.Event.(*helpers.GenericSlackEvent); ok {
		eventMap = genericEvent.RawData
		eventTypeStr = genericEvent.GetType()
	} else {
		// Fallback: try to marshal/unmarshal to get raw data
		if eventBytes, err := json.Marshal(eventArgs.Event); err == nil {
			_ = json.Unmarshal(eventBytes, &eventMap)
		}
		eventTypeStr = eventArgs.Event.GetType()
	}

	// Check event type constraint (string)
	if listener.constraints.eventType != "" {
		if eventTypeStr != listener.constraints.eventType {
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
	if listener.constraints.callbackID != "" && eventTypeStr == "function_executed" {
		if eventMap != nil {
			if function, exists := eventMap["function"]; exists {
				if functionMap, ok := function.(map[string]interface{}); ok {
					if callbackID, exists := functionMap["callback_id"]; exists {
						if callbackIDStr, ok := callbackID.(string); ok {
							return callbackIDStr == listener.constraints.callbackID
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
	if listener.constraints.actionType != "" {
		bodyMap, err := helpers.ExtractRawDataFromSlackAction(actionArgs.Body)
		if err != nil {
			return false
		}

		actionType, exists := bodyMap["type"]
		if !exists {
			return false
		}

		actionTypeStr, ok := actionType.(string)
		if !ok || actionTypeStr != listener.constraints.actionType {
			return false
		}
	}

	// If there are no specific field constraints, match on type only
	if listener.constraints.actionID == "" && listener.constraints.blockID == "" && listener.constraints.callbackID == "" &&
		listener.constraints.actionIDPattern == nil && listener.constraints.blockIDPattern == nil && listener.constraints.callbackIDPattern == nil {
		return true
	}

	actionMap, err := helpers.ExtractRawDataFromSlackAction(actionArgs.Action)
	if err != nil {
		return false
	}

	// Check action_id constraint (string or regexp)
	if listener.constraints.actionID != "" {
		actionID, exists := actionMap["action_id"]
		if !exists {
			return false
		}
		actionIDStr, ok := actionID.(string)
		if !ok {
			return false
		}
		if actionIDStr != listener.constraints.actionID {
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
	if listener.constraints.blockID != "" {
		blockID, exists := actionMap["block_id"]
		if !exists {
			return false
		}
		blockIDStr, ok := blockID.(string)
		if !ok {
			return false
		}
		if blockIDStr != listener.constraints.blockID {
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
	if listener.constraints.callbackID != "" {
		// Check in payload first
		if bodyMap, err := helpers.ExtractRawDataFromSlackAction(actionArgs.Body); err == nil {
			if callbackID, exists := bodyMap["callback_id"]; exists {
				callbackIDStr, ok := callbackID.(string)
				if ok && callbackIDStr == listener.constraints.callbackID {
					return true
				}
			}
		}
		return false
	} else if listener.constraints.callbackIDPattern != nil {
		// Check in payload first
		if bodyMap, err := helpers.ExtractRawDataFromSlackAction(actionArgs.Body); err == nil {
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
	if listener.constraints.command != "" {
		if commandArgs.Command.Command != listener.constraints.command {
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

	bodyMap, err := helpers.ExtractRawDataFromSlackShortcut(shortcutArgs.Body)
	if err != nil {
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
	if listener.constraints.callbackID != "" {
		if callbackIDStr != listener.constraints.callbackID {
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
	if listener.constraints.shortcutType != "" {
		shortcutType, exists := bodyMap["type"]
		if !exists {
			return false
		}
		shortcutTypeStr, ok := shortcutType.(string)
		if !ok || shortcutTypeStr != listener.constraints.shortcutType {
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

	bodyMap, err := helpers.ExtractRawDataFromSlackView(viewArgs.Body)
	if err != nil {
		return false
	}

	// Check view type constraint
	if listener.constraints.viewType != "" {
		viewType, exists := bodyMap["type"]
		if !exists {
			return false
		}
		viewTypeStr, ok := viewType.(string)
		if !ok || viewTypeStr != listener.constraints.viewType {
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
	if listener.constraints.callbackID != "" {
		if callbackIDStr != listener.constraints.callbackID {
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
	if listener.constraints.actionID != "" {
		if actionIDStr != listener.constraints.actionID {
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
	if listener.constraints.blockID != "" {
		blockID, exists := bodyMap["block_id"]
		if !exists {
			return false
		}
		blockIDStr, ok := blockID.(string)
		if !ok || blockIDStr != listener.constraints.blockID {
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

// Helper function to safely dereference string pointers
func getStringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
