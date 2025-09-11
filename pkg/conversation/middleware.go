package conversation

import (
	"time"

	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/types"
)

// UpdateConversationFunc represents a function to update conversation state
type UpdateConversationFunc func(conversation any, expiresAt *time.Time) error

// ConversationContext creates a conversation context middleware
// This middleware allows listeners (and other middleware) to store state related
// to the conversationId of an incoming event using the context.UpdateConversation()
// function. That state will be made available in future events that take place
// in the same conversation by reading from context.Conversation.
func ConversationContext(store ConversationStore) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Extract conversation ID from the request body
		var body []byte
		if args.Context.Custom != nil {
			if bodyBytes, exists := args.Context.Custom["body"]; exists {
				if bytes, ok := bodyBytes.([]byte); ok {
					body = bytes
				}
			}
		}

		if len(body) == 0 {
			args.Logger.Debug("No body available for conversation context")
			return args.Next()
		}

		typeAndConv := helpers.GetTypeAndConversation(body)

		if typeAndConv.ConversationID != nil {
			conversationID := *typeAndConv.ConversationID

			// Add update function to context
			args.Context.UpdateConversation = func(conversation any, expiresAt *time.Time) error {
				return store.Set(conversationID, conversation, expiresAt)
			}

			// Try to load existing conversation state
			if existingState, err := store.Get(conversationID); err == nil {
				args.Context.Conversation = existingState
				args.Logger.Debug("Conversation context loaded", "conversation_id", conversationID)
			} else {
				if err.Error() != "conversation not found" {
					// The conversation data can be expired - error: Conversation expired
					args.Logger.Debug("Conversation context failed loading", "conversation_id", conversationID, "error", err.Error())
				}
			}
		} else {
			args.Logger.Debug("No conversation ID for incoming event")
		}

		return args.Next()
	}
}
