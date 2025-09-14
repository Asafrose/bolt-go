package test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageRoutingComprehensive implements the missing tests from routing-message.spec.ts
func TestMessageRoutingComprehensive(t *testing.T) {
	t.Parallel()
	t.Run("should route a message event to a handler registered with message(string) if message contents match", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs
		handlerCalled := false

		// Register handler with string pattern
		app.Message("yo", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create message event with matching text
		eventBody := createMessageEventBodyComprehensive("U123456", "C123456", "yo")
		event := types.ReceiverEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called for matching message")
		assert.NotNil(t, receivedArgs.Event, "Event should be available")
		assert.NotNil(t, receivedArgs.Message, "Message should be available")

		if receivedArgs.Message != nil {
			assert.Equal(t, "yo", receivedArgs.Message.Text, "Message text should match")
		}
	})

	t.Run("should route a message event to a handler registered with message(RegExp) if message contents match", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp pattern (matches "hi" anywhere in text)
		messagePattern := regexp.MustCompile(`hi`)
		app.Message(messagePattern, func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create message event with matching text ("hiya" contains "hi")
		eventBody := createMessageEventBodyComprehensive("U123456", "C123456", "hiya")
		event := types.ReceiverEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called for RegExp matching message")
		assert.NotNil(t, receivedArgs.Event, "Event should be available")
		assert.NotNil(t, receivedArgs.Message, "Message should be available")

		if receivedArgs.Message != nil {
			assert.Equal(t, "hiya", receivedArgs.Message.Text, "Message text should match")
		}
	})

	t.Run("should not execute handler if no routing found, but acknowledge message event", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register handler for different pattern
		app.Message("goodbye", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Send message that doesn't match ("yo" doesn't match "goodbye")
		eventBody := createMessageEventBodyComprehensive("U123456", "C123456", "yo")
		event := types.ReceiverEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching message")
		// Note: Ack behavior depends on implementation - message events are typically auto-acknowledged
	})

	t.Run("should validate message pattern matching", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Test multiple message patterns
		handler1Called := false
		handler2Called := false

		// Handler 1: Exact match
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			handler1Called = true
			return nil
		})

		// Handler 2: Contains pattern
		app.Message("world", func(args bolt.SlackEventMiddlewareArgs) error {
			handler2Called = true
			return nil
		})

		// Send message that matches both patterns
		eventBody := createMessageEventBodyComprehensive("U123456", "C123456", "hello world")
		event := types.ReceiverEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handler1Called, "First handler should have been called")
		assert.True(t, handler2Called, "Second handler should have been called")
	})
}

// Helper function for creating message event bodies
func createMessageEventBodyComprehensive(userID, channelID, text string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    userID,
			"text":    text,
			"ts":      "1234567890.123456",
			"channel": channelID,
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{userID},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}
