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

// Test helpers for event routing
func createAppMentionEventBody() []byte {
	event := map[string]interface{}{
		"token":      "verification-token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "app_mention",
			"user":    "U123456",
			"text":    "<@U987654> hello",
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

func createMessageEventBody() []byte {
	event := map[string]interface{}{
		"token":      "verification-token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    "U123456",
			"text":    "hello world",
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

func TestAppEventRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route Slack event to handler registered with event type string", func(t *testing.T) {
		handlerCalled := false
		ackCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createAppMentionEventBody(),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				ackCalled = true
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Event handler should have been called")
		_ = ackCalled // Acknowledge variable is declared for future use
	})

	t.Run("should route message event to handler registered with message type", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			assert.NotNil(t, args.Event, "Event should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageEventBody(),
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

		assert.True(t, handlerCalled, "Message handler should have been called")
	})

	t.Run("should not execute handler if no routing found", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register handler for different event type
		app.Event("reaction_added", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Send app_mention event
		event := types.ReceiverEvent{
			Body: createAppMentionEventBody(),
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

		assert.False(t, handlerCalled, "Handler should not have been called for non-matching event")
	})

	t.Run("should handle multiple handlers for same event type", func(t *testing.T) {
		handler1Called := false
		handler2Called := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register multiple handlers for same event
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			handler1Called = true
			return nil
		})

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			handler2Called = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createAppMentionEventBody(),
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

		assert.True(t, handler1Called, "First handler should have been called")
		assert.True(t, handler2Called, "Second handler should have been called")
	})

	t.Run("should pass correct event data to handler", func(t *testing.T) {
		var receivedEvent interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register event handler that captures event data
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedEvent = args.Event
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createAppMentionEventBody(),
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

		assert.NotNil(t, receivedEvent, "Event data should have been passed to handler")

		// Verify event structure
		eventMap, ok := ExtractRawEventData(receivedEvent.(types.SlackEvent))
		require.True(t, ok, "Event should be extractable as map")
		assert.Equal(t, "app_mention", eventMap["type"], "Event type should be app_mention")
		assert.Equal(t, "U123456", eventMap["user"], "Event user should be correct")
	})
}
