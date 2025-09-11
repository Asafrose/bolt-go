package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssistantRouting implements the missing tests from routing-assistant.spec.ts
func TestAssistantRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route assistant_thread_started event to a registered handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var threadStartedCalled, threadContextChangedCalled, userMessageCalled bool

		config := bolt.AssistantConfig{
			ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
				func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
					threadStartedCalled = true
					assert.NotNil(t, args.Event, "Event should be available")
					assert.NotNil(t, args.Body, "Body should be available")
					assert.NotNil(t, args.GetThreadContext, "GetThreadContext should be available")
					return args.Next()
				},
			},
			ThreadContextChanged: []bolt.AssistantThreadContextChangedMiddleware{
				func(args bolt.AssistantThreadContextChangedMiddlewareArgs) error {
					threadContextChangedCalled = true
					return args.Next()
				},
			},
			UserMessage: []bolt.AssistantUserMessageMiddleware{
				func(args bolt.AssistantUserMessageMiddlewareArgs) error {
					userMessageCalled = true
					return args.Next()
				},
			},
		}

		assistant, err := bolt.NewAssistant(config)
		require.NoError(t, err)

		app.Assistant(assistant)

		// Create assistant_thread_started event
		eventBody := createAssistantThreadStartedEventBody("C123456", "1234567890.123456")
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

		assert.True(t, threadStartedCalled, "ThreadStarted handler should have been called")
		assert.False(t, threadContextChangedCalled, "ThreadContextChanged handler should NOT have been called")
		assert.False(t, userMessageCalled, "UserMessage handler should NOT have been called")
	})

	t.Run("should route assistant_thread_context_changed event to a registered handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var threadStartedCalled, threadContextChangedCalled, userMessageCalled bool

		config := bolt.AssistantConfig{
			ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
				func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
					threadStartedCalled = true
					return args.Next()
				},
			},
			ThreadContextChanged: []bolt.AssistantThreadContextChangedMiddleware{
				func(args bolt.AssistantThreadContextChangedMiddlewareArgs) error {
					threadContextChangedCalled = true
					assert.NotNil(t, args.Event, "Event should be available")
					assert.NotNil(t, args.Body, "Body should be available")
					assert.NotNil(t, args.GetThreadContext, "GetThreadContext should be available")
					return args.Next()
				},
			},
			UserMessage: []bolt.AssistantUserMessageMiddleware{
				func(args bolt.AssistantUserMessageMiddlewareArgs) error {
					userMessageCalled = true
					return args.Next()
				},
			},
		}

		assistant, err := bolt.NewAssistant(config)
		require.NoError(t, err)

		app.Assistant(assistant)

		// Create assistant_thread_context_changed event
		eventBody := createAssistantThreadContextChangedEventBody("C123456", "1234567890.123456")
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

		assert.False(t, threadStartedCalled, "ThreadStarted handler should NOT have been called")
		assert.True(t, threadContextChangedCalled, "ThreadContextChanged handler should have been called")
		assert.False(t, userMessageCalled, "UserMessage handler should NOT have been called")
	})

	t.Run("should route a message assistant scoped event to a registered handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var threadStartedCalled, threadContextChangedCalled, userMessageCalled bool

		config := bolt.AssistantConfig{
			ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
				func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
					threadStartedCalled = true
					return args.Next()
				},
			},
			ThreadContextChanged: []bolt.AssistantThreadContextChangedMiddleware{
				func(args bolt.AssistantThreadContextChangedMiddlewareArgs) error {
					threadContextChangedCalled = true
					return args.Next()
				},
			},
			UserMessage: []bolt.AssistantUserMessageMiddleware{
				func(args bolt.AssistantUserMessageMiddlewareArgs) error {
					userMessageCalled = true
					assert.NotNil(t, args.Event, "Event should be available")
					assert.NotNil(t, args.Body, "Body should be available")
					assert.NotNil(t, args.Message, "Message should be available")
					assert.NotNil(t, args.GetThreadContext, "GetThreadContext should be available")
					return args.Next()
				},
			},
		}

		assistant, err := bolt.NewAssistant(config)
		require.NoError(t, err)

		app.Assistant(assistant)

		// Create assistant user message event (message in assistant thread)
		eventBody := createAssistantUserMessageEventBody("U123456", "D123456", "1234567890.123456", "Hello assistant")
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

		assert.False(t, threadStartedCalled, "ThreadStarted handler should NOT have been called")
		assert.False(t, threadContextChangedCalled, "ThreadContextChanged handler should NOT have been called")
		assert.True(t, userMessageCalled, "UserMessage handler should have been called")
	})

	t.Run("should not execute handler if no routing found, but acknowledge event", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var handlerCalled bool

		config := bolt.AssistantConfig{
			ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
				func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
					handlerCalled = true
					return args.Next()
				},
			},
			UserMessage: []bolt.AssistantUserMessageMiddleware{
				func(args bolt.AssistantUserMessageMiddlewareArgs) error {
					handlerCalled = true
					return args.Next()
				},
			},
		}

		assistant, err := bolt.NewAssistant(config)
		require.NoError(t, err)

		app.Assistant(assistant)

		// Send non-assistant event (regular message, not in assistant thread)
		eventBody := createMessageEventBodyBuiltin("U123456", "C123456", "Hello world")
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-assistant event")
	})
}

// Helper functions for creating assistant event bodies

func createAssistantThreadStartedEventBody(channelID, threadTS string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":      "assistant_thread_started",
			"channel":   channelID,
			"thread_ts": threadTS,
			"user_id":   "U123456",
			"ts":        "1234567890.123456",
			"assistant_thread": map[string]interface{}{
				"user_id":           "U123456",
				"context":           map[string]interface{}{},
				"channel_id":        channelID,
				"thread_ts":         threadTS,
				"title":             "Assistant Thread",
				"status":            "in_progress",
				"suggested_prompts": []interface{}{},
			},
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{"U123456"},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}

func createAssistantThreadContextChangedEventBody(channelID, threadTS string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":      "assistant_thread_context_changed",
			"channel":   channelID,
			"thread_ts": threadTS,
			"user_id":   "U123456",
			"ts":        "1234567890.123456",
			"assistant_thread": map[string]interface{}{
				"user_id":           "U123456",
				"context":           map[string]interface{}{"key": "value"},
				"channel_id":        channelID,
				"thread_ts":         threadTS,
				"title":             "Assistant Thread",
				"status":            "in_progress",
				"suggested_prompts": []interface{}{},
			},
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{"U123456"},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}

func createAssistantUserMessageEventBody(userID, channelID, threadTS, text string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":         "message",
			"user":         userID,
			"text":         text,
			"ts":           "1234567890.123456",
			"channel":      channelID,
			"thread_ts":    threadTS,
			"channel_type": "im", // Assistant messages are in DMs
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{userID},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}
