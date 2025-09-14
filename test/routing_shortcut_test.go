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

// Test helpers for shortcut routing
func createGlobalShortcutBody(callbackID string) []byte {
	shortcut := map[string]interface{}{
		"type":        "shortcut",
		"token":       "verification-token",
		"action_ts":   "1234567890.123456",
		"team":        map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":        map[string]interface{}{"id": "U123456", "name": "testuser"},
		"callback_id": callbackID,
		"trigger_id":  "123456.123456.abcdef",
	}

	body, _ := json.Marshal(shortcut)
	return body
}

func createMessageShortcutBody(callbackID string) []byte {
	shortcut := map[string]interface{}{
		"type":        "message_action",
		"token":       "verification-token",
		"action_ts":   "1234567890.123456",
		"team":        map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":        map[string]interface{}{"id": "U123456", "name": "testuser"},
		"channel":     map[string]interface{}{"id": "C123456", "name": "general"},
		"callback_id": callbackID,
		"trigger_id":  "123456.123456.abcdef",
		"message": map[string]interface{}{
			"type": "message",
			"user": "U123456",
			"text": "test message",
			"ts":   "1234567890.123456",
		},
	}

	body, _ := json.Marshal(shortcut)
	return body
}

func TestAppShortcutRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route global shortcut by callback_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler
		callbackID := "test_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("test_shortcut"),
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

		assert.True(t, handlerCalled, "Shortcut handler should have been called")
	})

	t.Run("should route message shortcut by callback_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler
		callbackID := "message_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageShortcutBody("message_shortcut"),
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

		assert.True(t, handlerCalled, "Message shortcut handler should have been called")
	})

	t.Run("should route shortcut by type", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler by type
		shortcutType := "shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			Type: shortcutType,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("any_callback"),
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

		assert.True(t, handlerCalled, "Shortcut handler should have been called by type")
	})

	t.Run("should not route shortcut if callback_id doesn't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler with different callback_id
		callbackID := "different_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("test_shortcut"),
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

		assert.False(t, handlerCalled, "Shortcut handler should not have been called")
	})

	t.Run("should pass correct shortcut data to handler", func(t *testing.T) {
		var receivedShortcut interface{}
		var receivedBody interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler that captures shortcut data
		callbackID := "test_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedShortcut = args.Shortcut
			receivedBody = args.Body
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Ack, "Ack function should be present")
			// Say function should only be present for message shortcuts
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("test_shortcut"),
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

		assert.NotNil(t, receivedShortcut, "Shortcut data should have been passed to handler")
		assert.NotNil(t, receivedBody, "Body data should have been passed to handler")

		// Verify shortcut structure (simplified since we're using interface{})
		assert.NotNil(t, receivedShortcut, "Shortcut should be present")
	})

	t.Run("should handle message shortcut with Say function", func(t *testing.T) {
		var hasSayFunction bool

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message shortcut handler
		callbackID := "message_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			hasSayFunction = args.Say != nil
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createMessageShortcutBody("message_shortcut"),
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

		assert.True(t, hasSayFunction, "Message shortcuts should have Say function available")
	})

	t.Run("should handle multiple constraints", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler with multiple constraints
		callbackID := "test_shortcut"
		shortcutType := "shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
			Type:       shortcutType,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches both constraints
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("test_shortcut"),
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

		assert.True(t, handlerCalled, "Shortcut handler should have been called when all constraints match")
	})

	t.Run("should not match if type constraint fails", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register shortcut handler with wrong type constraint
		callbackID := "test_shortcut"
		shortcutType := "message_action"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: callbackID,
			Type:       shortcutType,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event for global shortcut (type: "shortcut")
		event := types.ReceiverEvent{
			Body: createGlobalShortcutBody("test_shortcut"),
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

		assert.False(t, handlerCalled, "Shortcut handler should not have been called when type constraint doesn't match")
	})
}
