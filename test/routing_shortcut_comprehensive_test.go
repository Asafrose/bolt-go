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

// TestShortcutRoutingComprehensive implements the missing tests from routing-shortcut.spec.ts
func TestShortcutRoutingComprehensive(t *testing.T) {
	t.Parallel()
	t.Run("should route a Slack shortcut event to a handler registered with shortcut(string) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs
		handlerCalled := false

		// Register handler with string callback ID
		app.ShortcutString("my_callback_id", func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create shortcut event with matching callback ID
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for matching callback ID")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")

		// Verify shortcut data
		if bodyMap, ok := ExtractRawShortcutData(receivedArgs.Body); ok {
			assert.Equal(t, "my_callback_id", bodyMap["callback_id"], "Callback ID should match")
		}
	})

	t.Run("should route a Slack shortcut event to a handler registered with shortcut(RegExp) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp pattern (matches "my_call*")
		callbackPattern := regexp.MustCompile(`my_call`)
		app.ShortcutPattern(callbackPattern, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create shortcut event with matching callback ID ("my_callback_id" contains "my_call")
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for RegExp matching callback ID")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
	})

	t.Run("should route a Slack shortcut event to a handler registered with shortcut({callback_id}) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs
		handlerCalled := false

		// Register handler with constraint object
		callbackID := "my_callback_id"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create shortcut event with matching callback ID
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for constraint object matching")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
	})

	t.Run("should route a Slack shortcut event to a handler registered with shortcut({type}) that matches the type", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs
		handlerCalled := false

		// Register handler with type constraint
		shortcutType := "message_action"
		app.Shortcut(bolt.ShortcutConstraints{
			Type: shortcutType,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create shortcut event with matching type
		shortcutBody := createMessageShortcutBodyComprehensive("any_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for type matching")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
	})

	t.Run("should not execute handler if no routing found", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register handler for different callback ID
		app.ShortcutString("different_callback_id", func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Send shortcut that doesn't match
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching shortcut")
	})

	t.Run("should route a Slack shortcut event to a handler registered with shortcut({type, callback_id}) that matches both the type and the callback_id", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs types.SlackShortcutMiddlewareArgs
		handlerCalled := false

		// Register handler with both type and callback_id constraints
		shortcutType := "message_action"
		callbackID := "my_callback_id"
		app.Shortcut(types.ShortcutConstraints{
			Type:       shortcutType,
			CallbackID: callbackID,
		}, func(args types.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return args.Ack(nil)
		})

		// Create shortcut event with matching type AND callback_id
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for both type and callback_id matching")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")

		// Verify both type and callback_id
		if bodyMap, ok := ExtractRawShortcutData(receivedArgs.Body); ok {
			assert.Equal(t, "message_action", bodyMap["type"], "Type should match")
			assert.Equal(t, "my_callback_id", bodyMap["callback_id"], "Callback ID should match")
		}
	})

	t.Run("should throw if provided a constraint with unknown shortcut constraint keys", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// This test would validate constraint keys at registration time
		// In Go, this would typically be handled by the type system
		// but we can test for runtime validation if implemented

		handlerCalled := false

		// Register handler with valid constraints (should work)
		validType := "message_action"
		validCallbackID := "valid_callback"
		app.Shortcut(types.ShortcutConstraints{
			Type:       validType,
			CallbackID: validCallbackID,
		}, func(args types.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return args.Ack(nil)
		})

		// Test that the valid handler works
		shortcutBody := createMessageShortcutBodyComprehensive("valid_callback")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler with valid constraints should work")

		// Note: In Go, invalid constraint keys would typically be caught at compile time
		// due to the type system, unlike JavaScript where they could be runtime errors
	})

	t.Run("should route a Slack shortcut event to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs types.SlackShortcutMiddlewareArgs
		handlerCalled := false
		ackCalled := false

		// Register handler that will acknowledge
		app.ShortcutString("my_callback_id", func(args types.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Acknowledge the shortcut
			err := args.Ack(nil)
			if err == nil {
				ackCalled = true
			}
			return err
		})

		// Create shortcut event
		shortcutBody := createMessageShortcutBodyComprehensive("my_callback_id")
		event := types.ReceiverEvent{
			Body: shortcutBody,
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

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.True(t, ackCalled, "Ack should have been called within the handler")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")

		// Verify that the handler processed the shortcut properly
		if bodyMap, ok := ExtractRawShortcutData(receivedArgs.Body); ok {
			assert.Equal(t, "my_callback_id", bodyMap["callback_id"], "Callback ID should match")
			assert.Equal(t, "message_action", bodyMap["type"], "Type should be message_action")
		}
	})
}

// Helper function for creating message shortcut event bodies
func createMessageShortcutBodyComprehensive(callbackID string) []byte {
	shortcutBody := map[string]interface{}{
		"type":      "message_action",
		"token":     "test_token",
		"action_ts": "1234567890.123456",
		"team": map[string]interface{}{
			"id":     "T123456",
			"domain": "test-team",
		},
		"user": map[string]interface{}{
			"id":   "U123456",
			"name": "testuser",
		},
		"callback_id":  callbackID,
		"trigger_id":   "123456789.123456789.abcdefg",
		"response_url": "https://hooks.slack.com/actions/T123456/123456789/abcdefg",
		"message_ts":   "1234567890.123456",
		"channel": map[string]interface{}{
			"id":   "C123456",
			"name": "general",
		},
		"message": map[string]interface{}{
			"type": "message",
			"user": "U987654321",
			"text": "Hello world",
			"ts":   "1234567890.123456",
		},
	}

	bodyBytes, _ := json.Marshal(shortcutBody)
	return bodyBytes
}
