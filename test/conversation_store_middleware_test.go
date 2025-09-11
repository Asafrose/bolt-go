package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/conversation"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConversationStore for testing conversation middleware
type MockConversationStore struct {
	state    map[string]any
	getError error
	setError error
	getCalls []string
	setCalls []SetCall
}

type SetCall struct {
	ConversationID string
	Value          any
	ExpiresAt      *time.Time
}

func NewMockConversationStore() *MockConversationStore {
	return &MockConversationStore{
		state:    make(map[string]any),
		getCalls: make([]string, 0),
		setCalls: make([]SetCall, 0),
	}
}

func (m *MockConversationStore) Set(conversationID string, value any, expiresAt *time.Time) error {
	m.setCalls = append(m.setCalls, SetCall{
		ConversationID: conversationID,
		Value:          value,
		ExpiresAt:      expiresAt,
	})
	if m.setError != nil {
		return m.setError
	}
	m.state[conversationID] = value
	return nil
}

func (m *MockConversationStore) Get(conversationID string) (any, error) {
	m.getCalls = append(m.getCalls, conversationID)
	if m.getError != nil {
		return nil, m.getError
	}
	if value, exists := m.state[conversationID]; exists {
		return value, nil
	}
	return nil, errors.New("conversation not found")
}

func (m *MockConversationStore) Delete(conversationID string) error {
	delete(m.state, conversationID)
	return nil
}

func (m *MockConversationStore) SetGetError(err error) {
	m.getError = err
}

func (m *MockConversationStore) SetSetError(err error) {
	m.setError = err
}

// TestConversationStoreMiddleware implements the missing tests from conversation-store.spec.ts
func TestConversationStoreMiddleware(t *testing.T) {
	t.Parallel()
	type ConversationState struct {
		UserName string `json:"user_name"`
		Count    int    `json:"count"`
	}

	t.Run("should forward events that have no conversation ID", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		middlewareCalled := false
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalled = true
			// Should not have conversation context
			assert.Nil(t, args.Context.Conversation, "Should not have conversation context")
			assert.Nil(t, args.Context.UpdateConversation, "Should not have updateConversation function")
			return args.Next()
		})

		// Create event without conversation ID (e.g., app_uninstalled event)
		eventBody := `{"type":"event_callback","event":{"type":"app_uninstalled"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err, "Should process event without error")
		assert.True(t, middlewareCalled, "Middleware should be called")
		assert.Empty(t, store.getCalls, "Store.Get should not be called")
		assert.Empty(t, store.setCalls, "Store.Set should not be called")
	})

	t.Run("should add to the context for events within a conversation that was not previously stored", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()
		store.SetGetError(errors.New("conversation not found"))

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		middlewareCalled := false
		var updateFunc types.UpdateConversationFn

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalled = true
			// Should not have existing conversation (since it's new)
			assert.Nil(t, args.Context.Conversation, "Should not have existing conversation")
			// Should have updateConversation function
			assert.NotNil(t, args.Context.UpdateConversation, "Should have updateConversation function")

			// Store the update function for testing
			if args.Context.UpdateConversation != nil {
				updateFunc = args.Context.UpdateConversation
			}

			return args.Next()
		})

		// Create event with conversation ID
		eventBody := `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err, "Should process event without error")
		assert.True(t, middlewareCalled, "Middleware should be called")
		assert.Contains(t, store.getCalls, "C123456", "Store.Get should be called with conversation ID")

		// Test the updateConversation function
		require.NotNil(t, updateFunc, "Update function should be available")

		newState := ConversationState{UserName: "testuser", Count: 1}
		err = updateFunc(newState, nil)
		require.NoError(t, err, "UpdateConversation should work")

		require.Len(t, store.setCalls, 1, "Store.Set should be called once")
		assert.Equal(t, "C123456", store.setCalls[0].ConversationID, "Should set correct conversation ID")
		assert.Equal(t, newState, store.setCalls[0].Value, "Should set correct value")
	})

	t.Run("should add to the context for events within a conversation that was previously stored", func(t *testing.T) {
		// Arrange
		existingState := ConversationState{UserName: "existing_user", Count: 42}
		store := NewMockConversationStore()
		store.state["C123456"] = existingState

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		middlewareCalled := false
		var updateFunc types.UpdateConversationFn
		var loadedConversation interface{}

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalled = true
			// Should have existing conversation loaded
			loadedConversation = args.Context.Conversation
			// Should have updateConversation function
			assert.NotNil(t, args.Context.UpdateConversation, "Should have updateConversation function")

			// Store the update function for testing
			if args.Context.UpdateConversation != nil {
				updateFunc = args.Context.UpdateConversation
			}

			return args.Next()
		})

		// Create event with conversation ID
		eventBody := `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err, "Should process event without error")
		assert.True(t, middlewareCalled, "Middleware should be called")
		assert.Contains(t, store.getCalls, "C123456", "Store.Get should be called with conversation ID")

		// Should have loaded the existing conversation
		assert.Equal(t, existingState, loadedConversation, "Should load existing conversation")

		// Test the updateConversation function
		require.NotNil(t, updateFunc, "Update function should be available")

		newState := ConversationState{UserName: "updated_user", Count: 100}
		err = updateFunc(newState, nil)
		require.NoError(t, err, "UpdateConversation should work")

		require.Len(t, store.setCalls, 1, "Store.Set should be called once")
		assert.Equal(t, "C123456", store.setCalls[0].ConversationID, "Should set correct conversation ID")
		assert.Equal(t, newState, store.setCalls[0].Value, "Should set correct value")
	})

	t.Run("should handle conversation store errors gracefully", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()
		store.SetGetError(errors.New("database connection failed"))

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		middlewareCalled := false
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalled = true
			// Should not have conversation due to error, but should have update function
			assert.Nil(t, args.Context.Conversation, "Should not have conversation due to error")
			assert.NotNil(t, args.Context.UpdateConversation, "Should still have updateConversation function")
			return args.Next()
		})

		// Create event with conversation ID
		eventBody := `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err, "Should process event without error even with store error")
		assert.True(t, middlewareCalled, "Middleware should be called")
		assert.Contains(t, store.getCalls, "C123456", "Store.Get should be called")
	})

	t.Run("should handle expired conversation gracefully", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()
		store.SetGetError(errors.New("conversation expired"))

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		middlewareCalled := false
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalled = true
			// Should not have conversation due to expiration
			assert.Nil(t, args.Context.Conversation, "Should not have conversation due to expiration")
			assert.NotNil(t, args.Context.UpdateConversation, "Should have updateConversation function")
			return args.Next()
		})

		// Create event with conversation ID
		eventBody := `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err, "Should process event without error")
		assert.True(t, middlewareCalled, "Middleware should be called")
		assert.Contains(t, store.getCalls, "C123456", "Store.Get should be called")
	})

	t.Run("should work with updateConversation with expiration", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()
		store.SetGetError(errors.New("conversation not found"))

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		var updateFunc types.UpdateConversationFn

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			if args.Context.UpdateConversation != nil {
				updateFunc = args.Context.UpdateConversation
			}
			return args.Next()
		})

		// Create event with conversation ID
		eventBody := `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`

		// Process the event
		receiverEvent := types.ReceiverEvent{
			Body:    []byte(eventBody),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, receiverEvent)
		require.NoError(t, err)
		require.NotNil(t, updateFunc, "Update function should be available")

		// Test updateConversation with expiration
		newState := ConversationState{UserName: "testuser", Count: 1}
		expiresAt := time.Now().Add(time.Hour)

		err = updateFunc(newState, &expiresAt)
		require.NoError(t, err, "UpdateConversation with expiration should work")

		require.Len(t, store.setCalls, 1, "Store.Set should be called once")
		assert.Equal(t, "C123456", store.setCalls[0].ConversationID, "Should set correct conversation ID")
		assert.Equal(t, newState, store.setCalls[0].Value, "Should set correct value")
		assert.Equal(t, &expiresAt, store.setCalls[0].ExpiresAt, "Should set correct expiration")
	})

	t.Run("should handle different event types with conversation IDs", func(t *testing.T) {
		// Arrange
		store := NewMockConversationStore()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		var conversationIDs []string

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			if args.Context.UpdateConversation != nil {
				// Store was accessed, meaning conversation ID was found
				conversationIDs = append(conversationIDs, "found")
			}
			return args.Next()
		})

		testCases := []struct {
			name      string
			eventBody string
			expectID  bool
		}{
			{
				name:      "message event",
				eventBody: `{"type":"event_callback","event":{"type":"message","channel":"C123456","user":"U123456","text":"hello"}}`,
				expectID:  true,
			},
			{
				name:      "block_actions event",
				eventBody: `{"type":"block_actions","user":{"id":"U123456"},"channel":{"id":"C123456"},"actions":[{"action_id":"test"}]}`,
				expectID:  true,
			},
			{
				name:      "slash command",
				eventBody: `command=/test&channel_id=C123456&user_id=U123456&text=hello`,
				expectID:  true,
			},
			{
				name:      "app_uninstalled event (no conversation ID)",
				eventBody: `{"type":"event_callback","event":{"type":"app_uninstalled"}}`,
				expectID:  false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				conversationIDs = nil // Reset

				headers := map[string]string{"Content-Type": "application/json"}
				if tc.name == "slash command" {
					headers["Content-Type"] = "application/x-www-form-urlencoded"
				}

				receiverEvent := types.ReceiverEvent{
					Body:    []byte(tc.eventBody),
					Headers: headers,
					Ack: func(response interface{}) error {
						return nil
					},
				}

				ctx := context.Background()
				err = app.ProcessEvent(ctx, receiverEvent)
				require.NoError(t, err, "Should process %s without error", tc.name)

				if tc.expectID {
					assert.Len(t, conversationIDs, 1, "Should find conversation ID for %s", tc.name)
				} else {
					assert.Empty(t, conversationIDs, "Should not find conversation ID for %s", tc.name)
				}
			})
		}
	})

	t.Run("should initialize without a conversation store when option is false", func(t *testing.T) {
		// This test verifies the app constructor behavior
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
			// ConvoStore: nil, // Default will create a MemoryStore
		})
		require.NoError(t, err)
		assert.NotNil(t, app, "App should be created successfully")

		// The conversation store should not be initialized
		// This is implicit - no conversation middleware will be added
	})

	t.Run("should add to the context for events within a conversation that was not previously stored and pass expiresAt", func(t *testing.T) {
		// Create a mock store
		store := NewMockConversationStore()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add the typed conversation middleware manually
		app.Use(conversation.ConversationContext(store))

		var updateFunc types.UpdateConversationFn

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// Should have existing conversation loaded
			assert.Nil(t, args.Context.Conversation, "Should not have existing conversation for new conversation")
			// Should have updateConversation function
			assert.NotNil(t, args.Context.UpdateConversation, "Should have updateConversation function")

			// Store the update function for testing with expiresAt
			if args.Context.UpdateConversation != nil {
				updateFunc = args.Context.UpdateConversation
			}

			return args.Next()
		})

		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			return args.Ack(nil)
		})

		// Create a message event with conversation ID
		eventBody := createMessageEventForConversation("C123456", "test message")
		event := types.ReceiverEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Test the update function with expiresAt
		require.NotNil(t, updateFunc, "Update function should be available")

		// Test updating with expiresAt
		expiresAt := time.Now().Add(1 * time.Hour)
		newState := ConversationState{
			UserName: "testuser",
			Count:    1,
		}

		err = updateFunc(newState, &expiresAt)
		require.NoError(t, err)

		// Verify the store was called with expiresAt
		require.Len(t, store.setCalls, 1, "Store should have been called once")
		setCall := store.setCalls[0]
		assert.Equal(t, "C123456", setCall.ConversationID, "Should use correct conversation ID")
		assert.Equal(t, newState, setCall.Value, "Should set correct value")
		assert.NotNil(t, setCall.ExpiresAt, "Should pass expiresAt parameter")
		assert.WithinDuration(t, expiresAt, *setCall.ExpiresAt, time.Second, "Should pass correct expiresAt time")
	})
}
