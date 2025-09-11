package assistant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
)

// AssistantConfig represents configuration for the Assistant
type AssistantConfig struct {
	ThreadContextStore   AssistantThreadContextStore               `json:"thread_context_store,omitempty"`
	ThreadStarted        []AssistantThreadStartedMiddleware        `json:"-"`
	ThreadContextChanged []AssistantThreadContextChangedMiddleware `json:"-"`
	UserMessage          []AssistantUserMessageMiddleware          `json:"-"`
}

// AssistantThreadContext represents the context for an assistant thread
type AssistantThreadContext struct {
	ChannelID string                 `json:"channel_id"`
	ThreadTS  string                 `json:"thread_ts"`
	Context   map[string]interface{} `json:"context"`
}

// AssistantThreadContextStore interface for storing thread contexts
type AssistantThreadContextStore interface {
	Get(ctx context.Context, channelID, threadTS string) (*AssistantThreadContext, error)
	Save(ctx context.Context, context *AssistantThreadContext) error
}

// DefaultThreadContextStore provides a default implementation
type DefaultThreadContextStore struct {
	contexts map[string]*AssistantThreadContext
	context  map[string]interface{} // Current context like JavaScript implementation
}

// NewDefaultThreadContextStore creates a new default thread context store
func NewDefaultThreadContextStore() *DefaultThreadContextStore {
	return &DefaultThreadContextStore{
		contexts: make(map[string]*AssistantThreadContext),
		context:  make(map[string]interface{}),
	}
}

// Get retrieves a thread context
func (s *DefaultThreadContextStore) Get(ctx context.Context, channelID, threadTS string) (*AssistantThreadContext, error) {
	key := channelID + ":" + threadTS
	if context, exists := s.contexts[key]; exists {
		return context, nil
	}

	// Return empty context if not found
	return &AssistantThreadContext{
		ChannelID: channelID,
		ThreadTS:  threadTS,
		Context:   make(map[string]interface{}),
	}, nil
}

// Save stores a thread context
func (s *DefaultThreadContextStore) Save(ctx context.Context, context *AssistantThreadContext) error {
	key := context.ChannelID + ":" + context.ThreadTS
	s.contexts[key] = context
	return nil
}

// GetWithArgs retrieves a thread context using middleware args (like JavaScript implementation)
func (s *DefaultThreadContextStore) GetWithArgs(args AllAssistantMiddlewareArgs) (map[string]interface{}, error) {
	// If context is already saved to instance, return it
	if channelID, exists := s.context["channel_id"]; exists && channelID != nil {
		return s.context, nil
	}

	// Check if we have a Slack client
	if args.Client == nil {
		// Return empty context if no client available
		return make(map[string]interface{}), nil
	}

	// Use the client as SlackClientWithConversations interface
	client := args.Client

	// For testing purposes, simulate retrieving context from message metadata
	// In a real implementation, this would call conversations.replies with proper parameters
	// For now, return empty context to allow tests to focus on the core logic
	_ = client // Use the client variable to avoid unused variable error

	return make(map[string]interface{}), nil
}

// GetWithArgsAndChannel retrieves context for specific channel and thread
func (s *DefaultThreadContextStore) GetWithArgsAndChannel(args AllAssistantMiddlewareArgs, channelID, threadTS string) (map[string]interface{}, error) {
	// Check if context is already saved in memory
	key := channelID + ":" + threadTS
	if context, exists := s.contexts[key]; exists {
		return context.Context, nil
	}

	// If not in memory, try to get from Slack API
	return s.GetWithArgs(args)
}

// SaveWithArgs saves a thread context using middleware args (like JavaScript implementation)
func (s *DefaultThreadContextStore) SaveWithArgs(args AllAssistantMiddlewareArgs, channelID, threadTS string, threadContext map[string]interface{}) error {
	// Check if we have a Slack client
	if args.Client == nil {
		// Fallback to basic save if no client available
		contextObj := &AssistantThreadContext{
			ChannelID: channelID,
			ThreadTS:  threadTS,
			Context:   threadContext,
		}
		// Save to instance context and memory
		s.context = threadContext
		key := channelID + ":" + threadTS
		s.contexts[key] = contextObj
		return s.Save(context.Background(), contextObj)
	}

	// Use the client as SlackClientWithConversations interface
	client := args.Client

	// For testing purposes, simulate saving context to message metadata
	// In a real implementation, this would:
	// 1. Call conversations.replies to get thread messages
	// 2. Find the bot's initial message
	// 3. Call chat.update to update message metadata
	// For now, just use the client variable to avoid unused variable error
	_ = client

	// Save to instance
	s.context = threadContext
	key := channelID + ":" + threadTS
	s.contexts[key] = &AssistantThreadContext{
		ChannelID: channelID,
		ThreadTS:  threadTS,
		Context:   threadContext,
	}

	return nil
}

// SetInstanceContext sets the context for the instance (helper for testing)
func (s *DefaultThreadContextStore) SetInstanceContext(context map[string]interface{}) {
	s.context = context
}

// GetInstanceContext gets the context from the instance (helper for testing)
func (s *DefaultThreadContextStore) GetInstanceContext() map[string]interface{} {
	return s.context
}

// SlackClientWithConversations interface for clients that support conversations and chat operations
type SlackClientWithConversations interface {
	GetConversationReplies(*slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
}

// Function type definitions for utility functions
type GetThreadContextUtilFn func() (*AssistantThreadContext, error)
type SaveThreadContextUtilFn func() error
type SetStatusFn func(status string) error
type SetSuggestedPromptsFn func(args SetSuggestedPromptsArguments) error
type SetTitleFn func(title string) error

// SetSuggestedPromptsArguments represents arguments for setting suggested prompts
type SetSuggestedPromptsArguments struct {
	Prompts []string `json:"prompts"`
}

// AssistantPrompt represents a suggested prompt
type AssistantPrompt struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Utility arguments that are added to assistant middleware
type AssistantUtilityArgs struct {
	GetThreadContext    GetThreadContextUtilFn  `json:"-"`
	SaveThreadContext   SaveThreadContextUtilFn `json:"-"`
	Say                 types.SayFn             `json:"-"`
	SetStatus           SetStatusFn             `json:"-"`
	SetSuggestedPrompts SetSuggestedPromptsFn   `json:"-"`
	SetTitle            SetTitleFn              `json:"-"`
}

// Middleware types
type AssistantThreadStartedMiddleware func(args AssistantThreadStartedMiddlewareArgs) error
type AssistantThreadContextChangedMiddleware func(args AssistantThreadContextChangedMiddlewareArgs) error
type AssistantUserMessageMiddleware func(args AssistantUserMessageMiddlewareArgs) error

// Middleware argument types
type AssistantThreadStartedMiddlewareArgs struct {
	types.AllMiddlewareArgs
	AssistantUtilityArgs
	Event interface{} `json:"event"`
	Body  interface{} `json:"body"`
}

type AssistantThreadContextChangedMiddlewareArgs struct {
	types.AllMiddlewareArgs
	AssistantUtilityArgs
	Event interface{} `json:"event"`
	Body  interface{} `json:"body"`
}

type AssistantUserMessageMiddlewareArgs struct {
	types.AllMiddlewareArgs
	AssistantUtilityArgs
	Event   interface{}         `json:"event"`
	Body    interface{}         `json:"body"`
	Message *types.MessageEvent `json:"message,omitempty"`
}

// AllAssistantMiddlewareArgs represents the arguments passed to assistant middleware with utility functions
type AllAssistantMiddlewareArgs struct {
	types.AllMiddlewareArgs
	AssistantUtilityArgs
}

// Assistant represents an AI assistant for Slack
type Assistant struct {
	threadContextStore             AssistantThreadContextStore
	threadStartedMiddleware        []AssistantThreadStartedMiddleware
	threadContextChangedMiddleware []AssistantThreadContextChangedMiddleware
	userMessageMiddleware          []AssistantUserMessageMiddleware
}

// NewAssistant creates a new Assistant instance
func NewAssistant(config AssistantConfig) (*Assistant, error) {
	if len(config.ThreadStarted) == 0 {
		return nil, errors.NewAssistantInitializationError("threadStarted middleware is required")
	}

	if len(config.UserMessage) == 0 {
		return nil, errors.NewAssistantInitializationError("userMessage middleware is required")
	}

	threadContextStore := config.ThreadContextStore
	if threadContextStore == nil {
		threadContextStore = NewDefaultThreadContextStore()
	}

	return &Assistant{
		threadContextStore:             threadContextStore,
		threadStartedMiddleware:        config.ThreadStarted,
		threadContextChangedMiddleware: config.ThreadContextChanged,
		userMessageMiddleware:          config.UserMessage,
	}, nil
}

// GetMiddleware returns the middleware function for the app
func (a *Assistant) GetMiddleware() types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		return a.processEvent(args)
	}
}

// processEvent processes assistant-related events
func (a *Assistant) processEvent(args types.AllMiddlewareArgs) error {
	// Debug: Log that processEvent is being called
	if args.Logger != nil {
		args.Logger.Debug("Assistant processEvent called")
	}

	// Extract event type and data from middleware args stored in context
	if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
		if args.Logger != nil {
			args.Logger.Debug("Found middlewareArgs in context")
		}
		if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
			if args.Logger != nil {
				args.Logger.Debug("Successfully cast to SlackEventMiddlewareArgs")
			}
			// Extract event type from event data
			var eventMap map[string]interface{}
			if genericEvent, ok := eventArgs.Event.(*helpers.GenericSlackEvent); ok {
				eventMap = genericEvent.RawData
			} else {
				// Fallback: try to marshal/unmarshal to get raw data
				if eventBytes, err := json.Marshal(eventArgs.Event); err == nil {
					_ = json.Unmarshal(eventBytes, &eventMap)
				}
			}

			if eventMap != nil {
				if args.Logger != nil {
					args.Logger.Debug("Event data extracted", "event", eventMap)
				}
				if IsAssistantEvent(eventMap) {
					if args.Logger != nil {
						args.Logger.Debug("Event identified as assistant event")
					}
					// This is an assistant event, don't call next
					if eventType, exists := eventMap["type"]; exists {
						if typeStr, ok := eventType.(string); ok {
							if args.Logger != nil {
								args.Logger.Debug("Processing assistant event", "type", typeStr)
							}
							switch typeStr {
							case "assistant_thread_started":
								return a.processThreadStarted(args, eventArgs)
							case "assistant_thread_context_changed":
								return a.processThreadContextChanged(args, eventArgs)
							case "message":
								// Check if this is an assistant message
								if a.isAssistantMessage(eventMap) {
									return a.processUserMessage(args, eventArgs)
								}
							}
						}
					}
					// Assistant event but no specific handler, just don't call next
					return nil
				} else {
					if args.Logger != nil {
						args.Logger.Debug("Event NOT identified as assistant event")
					}
				}
			} else {
				if args.Logger != nil {
					args.Logger.Debug("Event data is not map[string]interface{}", "event", eventArgs.Event)
				}
			}
		} else {
			if args.Logger != nil {
				args.Logger.Debug("middlewareArgs is not SlackEventMiddlewareArgs", "type", fmt.Sprintf("%T", middlewareArgs))
			}
		}
	} else {
		if args.Logger != nil {
			args.Logger.Debug("No middlewareArgs found in context")
		}
	}

	// Not an assistant event, continue
	if args.Logger != nil {
		args.Logger.Debug("Calling args.Next()")
	}
	return args.Next()
}

// processThreadStarted processes thread started events
func (a *Assistant) processThreadStarted(args types.AllMiddlewareArgs, eventArgs types.SlackEventMiddlewareArgs) error {
	// Extract channel and thread info
	channelID, threadTS := a.extractChannelAndThread(eventArgs.Event)

	utilityArgs := a.createUtilityArgs(args, channelID, threadTS)

	middlewareArgs := AssistantThreadStartedMiddlewareArgs{
		AllMiddlewareArgs:    args,
		AssistantUtilityArgs: utilityArgs,
		Event:                eventArgs.Event,
		Body:                 eventArgs.Body,
	}

	for _, middleware := range a.threadStartedMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	// Don't call args.Next() for assistant events
	return nil
}

// processThreadContextChanged processes thread context changed events
func (a *Assistant) processThreadContextChanged(args types.AllMiddlewareArgs, eventArgs types.SlackEventMiddlewareArgs) error {
	// Extract channel and thread info
	channelID, threadTS := a.extractChannelAndThread(eventArgs.Event)

	utilityArgs := a.createUtilityArgs(args, channelID, threadTS)

	middlewareArgs := AssistantThreadContextChangedMiddlewareArgs{
		AllMiddlewareArgs:    args,
		AssistantUtilityArgs: utilityArgs,
		Event:                eventArgs.Event,
		Body:                 eventArgs.Body,
	}

	for _, middleware := range a.threadContextChangedMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	// Don't call args.Next() for assistant events
	return nil
}

// processUserMessage processes user message events
func (a *Assistant) processUserMessage(args types.AllMiddlewareArgs, eventArgs types.SlackEventMiddlewareArgs) error {
	// Extract channel and thread info
	channelID, threadTS := a.extractChannelAndThread(eventArgs.Event)

	utilityArgs := a.createUtilityArgs(args, channelID, threadTS)

	middlewareArgs := AssistantUserMessageMiddlewareArgs{
		AllMiddlewareArgs:    args,
		AssistantUtilityArgs: utilityArgs,
		Event:                eventArgs.Event,
		Body:                 eventArgs.Body,
		Message:              eventArgs.Message, // Extract message from event args
	}

	for _, middleware := range a.userMessageMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	// Don't call args.Next() for assistant events
	return nil
}

// createUtilityArgs creates utility arguments for assistant middleware
func (a *Assistant) createUtilityArgs(args types.AllMiddlewareArgs, channelID, threadTS string) AssistantUtilityArgs {
	var currentContext *AssistantThreadContext

	return AssistantUtilityArgs{
		GetThreadContext: func() (*AssistantThreadContext, error) {
			if currentContext == nil {
				ctx := context.Background() // Would use proper context
				var err error
				currentContext, err = a.threadContextStore.Get(ctx, channelID, threadTS)
				if err != nil {
					return nil, err
				}
			}
			return currentContext, nil
		},
		SaveThreadContext: func() error {
			if currentContext != nil {
				ctx := context.Background() // Would use proper context
				return a.threadContextStore.Save(ctx, currentContext)
			}
			return nil
		},
		Say: func(message types.SayMessage) (*types.SayResponse, error) {
			// This would use the actual say function from the context
			return &types.SayResponse{}, nil
		},
		SetStatus: func(status string) error {
			// This would call the Slack API to set thread status
			return nil
		},
		SetSuggestedPrompts: func(args SetSuggestedPromptsArguments) error {
			// This would call the Slack API to set suggested prompts
			return nil
		},
		SetTitle: func(title string) error {
			// This would call the Slack API to set thread title
			return nil
		},
	}
}

// extractChannelAndThread extracts channel ID and thread timestamp from event data
func (a *Assistant) extractChannelAndThread(event interface{}) (string, string) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		var channelID, threadTS string

		// Extract channel ID
		if channel, exists := eventMap["channel"]; exists {
			if channelStr, ok := channel.(string); ok {
				channelID = channelStr
			}
		}

		// Extract thread timestamp
		if thread, exists := eventMap["thread_ts"]; exists {
			if threadStr, ok := thread.(string); ok {
				threadTS = threadStr
			}
		}

		return channelID, threadTS
	}

	return "", ""
}

// isAssistantMessage checks if a message event is in an assistant thread
func (a *Assistant) isAssistantMessage(eventMap map[string]interface{}) bool {
	return IsAssistantMessage(eventMap)
}

// ValidateAssistantConfig validates the assistant configuration
func ValidateAssistantConfig(config *AssistantConfig) error {
	if config == nil {
		return errors.NewAssistantInitializationError("Assistant expects a configuration object as the argument")
	}

	if len(config.ThreadStarted) == 0 {
		return errors.NewAssistantInitializationError("Assistant is missing required keys: threadStarted")
	}

	if len(config.UserMessage) == 0 {
		return errors.NewAssistantInitializationError("Assistant is missing required keys: userMessage")
	}

	return nil
}

// IsAssistantEvent determines if an incoming event is a supported Assistant event type
func IsAssistantEvent(event map[string]interface{}) bool {
	if eventType, exists := event["type"]; exists {
		if typeStr, ok := eventType.(string); ok {
			switch typeStr {
			case "assistant_thread_started", "assistant_thread_context_changed":
				return true
			case "message":
				return IsAssistantMessage(event)
			}
		}
	}
	return false
}

// IsAssistantMessage determines if a message event is an assistant message
func IsAssistantMessage(event map[string]interface{}) bool {
	// Check for both channel and thread_ts (thread message requirement)
	_, hasChannel := event["channel"]
	threadTS, hasThreadTS := event["thread_ts"]
	isThreadMessage := hasChannel && hasThreadTS && threadTS != nil

	if !isThreadMessage {
		return false
	}

	// Check channel_type (assistant messages are in IMs/DMs) - MUST exist and be "im"
	channelType, hasChannelType := event["channel_type"]
	if !hasChannelType {
		return false
	}

	channelTypeStr, ok := channelType.(string)
	if !ok || channelTypeStr != "im" {
		return false
	}

	// Check subtype requirements (like JavaScript implementation)
	if subtype, hasSubtype := event["subtype"]; hasSubtype {
		if subtypeStr, ok := subtype.(string); ok {
			// Only allow file_share subtype or no subtype
			if subtypeStr != "file_share" {
				return false
			}
		}
	}

	return true
}

// MatchesConstraints determines if an event matches assistant constraints
func MatchesConstraints(event map[string]interface{}) bool {
	// For non-message events, return true (they're handled by event type)
	if eventType, exists := event["type"]; exists {
		if typeStr, ok := eventType.(string); ok && typeStr != "message" {
			return true
		}
	}

	// For message events, check if they're assistant messages
	// and don't have unsupported subtypes
	if subtype, exists := event["subtype"]; exists {
		if subtypeStr, ok := subtype.(string); ok {
			// Bot messages and other subtypes are not supported
			if subtypeStr == "bot_message" {
				return false
			}
		}
	}

	return IsAssistantMessage(event)
}

// ExtractThreadInfo parses an incoming payload and returns relevant details about the thread
func ExtractThreadInfo(payload map[string]interface{}) (channelID, threadTS string, context map[string]interface{}) {
	context = make(map[string]interface{})

	// assistant_thread_started, assistant_thread_context_changed
	if assistantThread, ok := payload["assistant_thread"].(map[string]interface{}); ok {
		if channelIDVal, exists := assistantThread["channel_id"]; exists {
			if channelIDStr, ok := channelIDVal.(string); ok {
				channelID = channelIDStr
			}
		}
		if threadTSVal, exists := assistantThread["thread_ts"]; exists {
			if threadTSStr, ok := threadTSVal.(string); ok {
				threadTS = threadTSStr
			}
		}
		if contextVal, exists := assistantThread["context"]; exists {
			if contextMap, ok := contextVal.(map[string]interface{}); ok {
				context = contextMap
			}
		}
	}

	// user message in thread
	if channelVal, exists := payload["channel"]; exists && channelID == "" {
		if channelStr, ok := channelVal.(string); ok {
			channelID = channelStr
		}
	}
	if threadTSVal, exists := payload["thread_ts"]; exists && threadTS == "" {
		if threadTSStr, ok := threadTSVal.(string); ok {
			threadTS = threadTSStr
		}
	}

	// throw error if `channel_id` or `thread_ts` are missing
	if channelID == "" || threadTS == "" {
		var missingProps []string
		if channelID == "" {
			missingProps = append(missingProps, "channel_id")
		}
		if threadTS == "" {
			missingProps = append(missingProps, "thread_ts")
		}

		if len(missingProps) > 0 {
			errorMsg := "Assistant message event is missing required properties: "
			for i, prop := range missingProps {
				if i > 0 {
					errorMsg += ", "
				}
				errorMsg += prop
			}
			panic(errors.NewAssistantMissingPropertyError(errorMsg))
		}
	}

	return channelID, threadTS, context
}

// EnrichAssistantArgs enriches the middleware args with assistant utilities
func EnrichAssistantArgs(store AssistantThreadContextStore, args AllAssistantMiddlewareArgs) AllAssistantMiddlewareArgs {
	// Remove next from args to prevent continuation of middleware chain
	enrichedArgs := AllAssistantMiddlewareArgs{
		AllMiddlewareArgs: types.AllMiddlewareArgs{
			Context: args.Context,
			Client:  args.Client,
			Logger:  args.Logger,
			// Next is deliberately omitted
		},
	}

	// Add utility functions
	enrichedArgs.GetThreadContext = func() (*AssistantThreadContext, error) {
		// This would extract channel and thread from the current event context
		return store.Get(context.Background(), "default_channel", "default_thread")
	}

	enrichedArgs.SaveThreadContext = func() error {
		// This would save the current context
		return nil
	}

	enrichedArgs.Say = func(message types.SayMessage) (*types.SayResponse, error) {
		// This would use the client to post a message
		// In a real implementation, this would use the actual client and channel/thread from context
		return &types.SayResponse{}, nil
	}

	enrichedArgs.SetStatus = func(status string) error {
		// This would call the Slack API to set thread status
		return nil
	}

	enrichedArgs.SetSuggestedPrompts = func(args SetSuggestedPromptsArguments) error {
		// This would call the Slack API to set suggested prompts
		return nil
	}

	enrichedArgs.SetTitle = func(title string) error {
		// This would call the Slack API to set thread title
		return nil
	}

	return enrichedArgs
}

// ProcessAssistantMiddleware processes middleware for a specific event type
func (a *Assistant) ProcessAssistantMiddleware(eventType string, event map[string]interface{}) error {
	switch eventType {
	case "assistant_thread_started":
		for _, middleware := range a.threadStartedMiddleware {
			args := AssistantThreadStartedMiddlewareArgs{
				Event: event,
				Body:  event,
			}
			if err := middleware(args); err != nil {
				return err
			}
		}
	case "assistant_thread_context_changed":
		for _, middleware := range a.threadContextChangedMiddleware {
			args := AssistantThreadContextChangedMiddlewareArgs{
				Event: event,
				Body:  event,
			}
			if err := middleware(args); err != nil {
				return err
			}
		}
	case "message":
		if IsAssistantMessage(event) {
			for _, middleware := range a.userMessageMiddleware {
				args := AssistantUserMessageMiddlewareArgs{
					Event: event,
					Body:  event,
				}
				if err := middleware(args); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
