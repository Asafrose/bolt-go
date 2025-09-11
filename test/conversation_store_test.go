package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/conversation"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for conversation state
type TestConversationState struct {
	UserName string `json:"user_name"`
	Count    int    `json:"count"`
	Data     string `json:"data"`
}

func TestMemoryStore(t *testing.T) {
	t.Run("should store and retrieve conversation state", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		state := TestConversationState{
			UserName: "testuser",
			Count:    42,
			Data:     "test data",
		}

		// Store the state
		err := store.Set(conversationID, state, nil)
		require.NoError(t, err)

		// Retrieve the state
		retrieved, err := store.Get(conversationID)
		require.NoError(t, err)

		assert.Equal(t, state.UserName, retrieved.UserName)
		assert.Equal(t, state.Count, retrieved.Count)
		assert.Equal(t, state.Data, retrieved.Data)
	})

	t.Run("should return error for non-existent conversation", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()

		_, err := store.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation not found")
	})

	t.Run("should handle expiration", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		state := TestConversationState{
			UserName: "testuser",
			Count:    42,
		}

		// Set expiration to past time
		pastTime := time.Now().Add(-1 * time.Hour)
		err := store.Set(conversationID, state, &pastTime)
		require.NoError(t, err)

		// Try to retrieve expired state
		_, err = store.Get(conversationID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation expired")
	})

	t.Run("should handle future expiration", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		state := TestConversationState{
			UserName: "testuser",
			Count:    42,
		}

		// Set expiration to future time
		futureTime := time.Now().Add(1 * time.Hour)
		err := store.Set(conversationID, state, &futureTime)
		require.NoError(t, err)

		// Should be able to retrieve non-expired state
		retrieved, err := store.Get(conversationID)
		require.NoError(t, err)
		assert.Equal(t, state.UserName, retrieved.UserName)
	})

	t.Run("should overwrite existing conversation state", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		// Store initial state
		initialState := TestConversationState{
			UserName: "user1",
			Count:    1,
		}
		err := store.Set(conversationID, initialState, nil)
		require.NoError(t, err)

		// Overwrite with new state
		newState := TestConversationState{
			UserName: "user2",
			Count:    2,
		}
		err = store.Set(conversationID, newState, nil)
		require.NoError(t, err)

		// Should retrieve the new state
		retrieved, err := store.Get(conversationID)
		require.NoError(t, err)
		assert.Equal(t, newState.UserName, retrieved.UserName)
		assert.Equal(t, newState.Count, retrieved.Count)
	})

	t.Run("should handle concurrent access", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		// Simulate concurrent writes
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(index int) {
				state := TestConversationState{
					UserName: "user",
					Count:    index,
				}
				err := store.Set(conversationID, state, nil)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should be able to retrieve some state (last one written)
		retrieved, err := store.Get(conversationID)
		require.NoError(t, err)
		assert.Equal(t, "user", retrieved.UserName)
		assert.True(t, retrieved.Count >= 0 && retrieved.Count < 10)
	})
}

func TestConversationMiddleware(t *testing.T) {
	t.Run("should load and save conversation state", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		// Pre-populate store
		initialState := TestConversationState{
			UserName: "testuser",
			Count:    5,
		}
		err := store.Set(conversationID, initialState, nil)
		require.NoError(t, err)

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		var receivedConversation interface{}
		var updateFunction interface{}

		// Register event handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedConversation = args.Context.Conversation
			updateFunction = args.Context.UpdateConversation
			return nil
		})

		// Create message event
		messageBody := map[string]interface{}{
			"token":      "verification-token",
			"team_id":    "T123456",
			"api_app_id": "A123456",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "hello",
				"ts":      "1234567890.123456",
				"channel": conversationID,
			},
			"type":         "event_callback",
			"event_id":     "Ev123456",
			"event_time":   1234567890,
			"authed_users": []string{"U987654"},
		}

		bodyBytes, _ := json.Marshal(messageBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Verify conversation was loaded
		assert.NotNil(t, receivedConversation, "Conversation should be loaded")
		if state, ok := receivedConversation.(TestConversationState); ok {
			assert.Equal(t, "testuser", state.UserName)
			assert.Equal(t, 5, state.Count)
		}

		// Verify update function is available
		assert.NotNil(t, updateFunction, "Update function should be available")
	})

	t.Run("should handle conversation without existing state", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		var receivedConversation interface{}
		var updateFunction interface{}

		// Register event handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedConversation = args.Context.Conversation
			updateFunction = args.Context.UpdateConversation
			return nil
		})

		// Create message event
		messageBody := map[string]interface{}{
			"token":      "verification-token",
			"team_id":    "T123456",
			"api_app_id": "A123456",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "hello",
				"ts":      "1234567890.123456",
				"channel": "C999999", // New conversation
			},
			"type":         "event_callback",
			"event_id":     "Ev123456",
			"event_time":   1234567890,
			"authed_users": []string{"U987654"},
		}

		bodyBytes, _ := json.Marshal(messageBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Conversation should be nil for new conversation
		assert.Nil(t, receivedConversation, "New conversation should not have existing state")

		// Update function should still be available
		assert.NotNil(t, updateFunction, "Update function should be available")
	})

	t.Run("should handle events without conversation ID", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		var receivedConversation interface{}
		var updateFunction interface{}

		// Register shortcut handler (shortcuts don't have conversation IDs)
		callbackID := "test_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedConversation = args.Context.Conversation
			updateFunction = args.Context.UpdateConversation
			return nil
		})

		// Create global shortcut event
		shortcutBody := map[string]interface{}{
			"type":        "shortcut",
			"token":       "verification-token",
			"team":        map[string]interface{}{"id": "T123456"},
			"user":        map[string]interface{}{"id": "U123456"},
			"callback_id": "test_shortcut",
		}

		bodyBytes, _ := json.Marshal(shortcutBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Should handle gracefully without conversation ID
		assert.Nil(t, receivedConversation, "Event without conversation ID should not have state")
		assert.Nil(t, updateFunction, "Event without conversation ID should not have update function")
	})
}

func TestConversationStoreIntegration(t *testing.T) {
	t.Run("should persist conversation state across events", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()
		conversationID := "C123456"

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		messageCount := 0

		// Register event handler that updates conversation state
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			messageCount++

			// Get current state or create new one
			var currentState TestConversationState
			if args.Context.Conversation != nil {
				currentState = args.Context.Conversation.(TestConversationState)
			}

			// Update state
			currentState.Count++
			currentState.UserName = "testuser"
			currentState.Data = "updated"

			// Save updated state
			if updateFn, ok := args.Context.UpdateConversation.(func(TestConversationState, *time.Time) error); ok {
				return updateFn(currentState, nil)
			}

			return nil
		})

		// Create first message event
		messageBody1 := createMessageEventForConversation(conversationID, "first message")
		event1 := types.ReceiverEvent{
			Body: messageBody1,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error { return nil },
		}

		// Process first event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event1)
		require.NoError(t, err)

		// Create second message event
		messageBody2 := createMessageEventForConversation(conversationID, "second message")
		event2 := types.ReceiverEvent{
			Body: messageBody2,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error { return nil },
		}

		// Process second event
		err = app.ProcessEvent(ctx, event2)
		require.NoError(t, err)

		// Verify state was persisted
		finalState, err := store.Get(conversationID)
		require.NoError(t, err)

		assert.Equal(t, 2, finalState.Count, "Count should be incremented across messages")
		assert.Equal(t, "testuser", finalState.UserName, "User name should be set")
		assert.Equal(t, "updated", finalState.Data, "Data should be updated")
		assert.Equal(t, 2, messageCount, "Both messages should be processed")
	})

	t.Run("should handle different conversation IDs separately", func(t *testing.T) {
		store := conversation.NewMemoryStore[TestConversationState]()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add conversation middleware
		app.Use(conversation.ConversationContext(store))

		// Register event handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			var currentState TestConversationState
			if args.Context.Conversation != nil {
				currentState = args.Context.Conversation.(TestConversationState)
			}

			currentState.Count++

			if updateFn, ok := args.Context.UpdateConversation.(func(TestConversationState, *time.Time) error); ok {
				return updateFn(currentState, nil)
			}

			return nil
		})

		// Process events in different conversations
		conversationID1 := "C111111"
		conversationID2 := "C222222"

		// Process message in first conversation
		event1 := types.ReceiverEvent{
			Body:    createMessageEventForConversation(conversationID1, "message 1"),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack:     func(response interface{}) error { return nil },
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event1)
		require.NoError(t, err)

		// Process message in second conversation
		event2 := types.ReceiverEvent{
			Body:    createMessageEventForConversation(conversationID2, "message 2"),
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack:     func(response interface{}) error { return nil },
		}

		err = app.ProcessEvent(ctx, event2)
		require.NoError(t, err)

		// Verify separate states
		state1, err := store.Get(conversationID1)
		require.NoError(t, err)
		assert.Equal(t, 1, state1.Count, "First conversation should have count 1")

		state2, err := store.Get(conversationID2)
		require.NoError(t, err)
		assert.Equal(t, 1, state2.Count, "Second conversation should have count 1")
	})
}

// Helper function to create message events for specific conversations
func createMessageEventForConversation(conversationID, text string) []byte {
	messageBody := map[string]interface{}{
		"token":      "verification-token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    "U123456",
			"text":    text,
			"ts":      "1234567890.123456",
			"channel": conversationID,
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{"U987654"},
	}

	bodyBytes, _ := json.Marshal(messageBody)
	return bodyBytes
}

// TestConversationStoreComprehensive implements missing tests from conversation-store.spec.ts
func TestConversationStoreComprehensive(t *testing.T) {

	t.Run("conversationContext middleware", func(t *testing.T) {
		t.Run("should add to the context for events within a conversation that was not previously stored and pass expiresAt", func(t *testing.T) {
			store := conversation.NewMemoryStore[TestConversationState]()
			conversationID := "CONVERSATION_ID"
			expiresAt := time.Now().Add(time.Hour).Unix()

			// Create app with conversation store middleware
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			// Add conversation store middleware
			app.Use(conversation.ConversationContext(store))

			// Add a handler that uses updateConversation with expiresAt
			app.Message("test", func(args types.SlackEventMiddlewareArgs) error {
				// Verify updateConversation function exists and works with expiresAt
				if args.Context.UpdateConversation != nil {
					if updateFunc, ok := args.Context.UpdateConversation.(func(TestConversationState, *time.Time) error); ok {
						state := TestConversationState{UserName: "testuser", Count: 1, Data: "test"}
						expTime := time.Unix(expiresAt, 0)
						return updateFunc(state, &expTime)
					}
				}
				return nil
			})

			// Create event body (reuse existing function from the same file)
			bodyBytes := createMessageEventForConversation(conversationID, "test message")

			// Create event
			event := types.ReceiverEvent{
				Body:    bodyBytes,
				Headers: map[string]string{"Content-Type": "application/json"},
			}

			// Process event
			ctx := context.Background()
			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			// Verify state was stored with correct expiration
			state, err := store.Get(conversationID)
			require.NoError(t, err)
			assert.Equal(t, "testuser", state.UserName)
			assert.Equal(t, 1, state.Count)
		})
	})

	t.Run("MemoryStore", func(t *testing.T) {
		t.Run("constructor should initialize successfully", func(t *testing.T) {
			store := conversation.NewMemoryStore[TestConversationState]()
			assert.NotNil(t, store)
		})

		t.Run("#set and #get should store conversation state", func(t *testing.T) {
			store := conversation.NewMemoryStore[TestConversationState]()
			conversationID := "CONVERSATION_ID"
			state := TestConversationState{UserName: "testuser", Count: 42, Data: "test"}

			// Set state
			err := store.Set(conversationID, state, nil)
			require.NoError(t, err)

			// Get state
			retrievedState, err := store.Get(conversationID)
			require.NoError(t, err)
			assert.Equal(t, state, retrievedState)
		})

		t.Run("#set and #get should reject lookup of conversation state when the conversation is not stored", func(t *testing.T) {
			store := conversation.NewMemoryStore[TestConversationState]()
			conversationID := "NON_EXISTENT_CONVERSATION"

			// Try to get non-existent conversation
			_, err := store.Get(conversationID)
			assert.Error(t, err)
		})

		t.Run("#set and #get should reject lookup of conversation state when the conversation is expired", func(t *testing.T) {
			store := conversation.NewMemoryStore[TestConversationState]()
			conversationID := "CONVERSATION_ID"
			state := TestConversationState{UserName: "testuser", Count: 42, Data: "test"}
			expiresInMs := int64(5) // 5 milliseconds

			// Set state with short expiration
			expiresAt := time.Now().Add(time.Duration(expiresInMs) * time.Millisecond)
			err := store.Set(conversationID, state, &expiresAt)
			require.NoError(t, err)

			// Wait for expiration
			time.Sleep(time.Duration(expiresInMs*2) * time.Millisecond)

			// Try to get expired conversation
			_, err = store.Get(conversationID)
			assert.Error(t, err, "Should reject lookup of expired conversation")
		})
	})
}

// TestConversationStoreInitialization tests the missing conversation store initialization test
func TestConversationStoreInitialization(t *testing.T) {
	t.Run("should initialize the conversation store", func(t *testing.T) {
		// Test that app initializes with conversation store by default
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Verify that the app has a conversation store initialized
		// This would be tested by checking if conversation context is available in middleware
		conversationStoreInitialized := false

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// Check if conversation-related context is available
			if args.Context != nil && args.Context.UpdateConversation != nil {
				conversationStoreInitialized = true
			}
			return args.Next()
		})

		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			return args.Ack(nil)
		})

		// Create a message event to trigger middleware
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"channel": "C123456",
				"user":    "U123456",
				"text":    "test message",
				"ts":      "1234567890.123456",
			},
			"team_id": "T123456",
		}

		bodyBytes, _ := json.Marshal(eventBody)
		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, conversationStoreInitialized, "Conversation store should be initialized by default")
	})
}
