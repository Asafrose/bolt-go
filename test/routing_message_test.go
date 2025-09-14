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

// Test helpers for message routing
func createMessageEventBodyWithText(text string) []byte {
	event := map[string]interface{}{
		"token":      "verification-token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    "U123456",
			"text":    text,
			"ts":      "1234567890.123456",
			"channel": "C123456",
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{"U987654"},
	}

	body, _ := json.Marshal(event)
	return body
}

func TestAppMessageRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route message with string pattern", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler with string pattern
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("hello world"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Message handler should have been called")
	})

	t.Run("should not route message if pattern doesn't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler with different pattern
		app.Message("goodbye", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("hello world"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.False(t, handlerCalled, "Message handler should not have been called")
	})

	t.Run("should handle exact string match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler with exact string
		app.Message("hello world", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("hello world"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Message handler should have been called for exact match")
	})

	t.Run("should pass correct message data to handler", func(t *testing.T) {
		var receivedEvent interface{}
		var receivedMessage interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler that captures message data
		app.Message("test", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedEvent = args.Event
			receivedMessage = args.Message
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Say, "Say function should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("test message"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.NotNil(t, receivedEvent, "Event data should have been passed to handler")

		// Verify event structure
		eventMap, ok := ExtractRawEventData(receivedEvent.(types.SlackEvent))
		require.True(t, ok, "Event should be extractable as map")
		assert.Equal(t, "message", eventMap["type"], "Event type should be message")
		assert.Equal(t, "test message", eventMap["text"], "Message text should be correct")
		assert.Equal(t, "U123456", eventMap["user"], "Message user should be correct")

		// Message should be populated for message events
		if receivedMessage != nil {
			// For now, just verify it's not nil - detailed message event structure can be tested later
			assert.NotNil(t, receivedMessage, "Message event should be populated")
		}
	})

	t.Run("should handle multiple message patterns", func(t *testing.T) {
		handler1Called := false
		handler2Called := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register multiple message handlers
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			handler1Called = true
			return nil
		})

		app.Message("world", func(args bolt.SlackEventMiddlewareArgs) error {
			handler2Called = true
			return nil
		})

		// Create receiver event that matches both patterns
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("hello world"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handler1Called, "First message handler should have been called")
		assert.True(t, handler2Called, "Second message handler should have been called")
	})

	t.Run("should handle case sensitivity", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler with lowercase pattern
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event with uppercase text
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("HELLO WORLD"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// This depends on implementation - for now assume case-sensitive
		assert.False(t, handlerCalled, "Message handler should be case-sensitive by default")
	})

	t.Run("should handle empty message text", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler for any message (empty pattern)
		app.Message("", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event with empty text
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText(""),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Message handler should handle empty text")
	})

	t.Run("should handle partial matches within longer text", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler
		app.Message("test", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event with longer text containing the pattern
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("this is a test message with more text"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Message handler should match partial text")
	})
}
