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

// Test helpers for action routing
func createButtonActionBody() []byte {
	action := map[string]interface{}{
		"type":        "interactive_message",
		"token":       "verification-token",
		"team":        map[string]interface{}{"id": "T123456"},
		"user":        map[string]interface{}{"id": "U123456"},
		"channel":     map[string]interface{}{"id": "C123456"},
		"callback_id": "button_callback",
		"actions": []interface{}{
			map[string]interface{}{
				"action_id": "button_1",
				"block_id":  "block_1",
				"type":      "button",
				"text":      map[string]interface{}{"type": "plain_text", "text": "Click me"},
				"value":     "button_value",
			},
		},
		"response_url": "https://hooks.slack.com/actions/T123456/123456/abcdef",
		"trigger_id":   "123456.123456.abcdef",
	}

	body, _ := json.Marshal(action)
	return body
}

func createBlockActionBody(actionID, blockID string) []byte {
	action := map[string]interface{}{
		"type":    "block_actions",
		"token":   "verification-token",
		"team":    map[string]interface{}{"id": "T123456"},
		"user":    map[string]interface{}{"id": "U123456"},
		"channel": map[string]interface{}{"id": "C123456"},
		"actions": []interface{}{
			map[string]interface{}{
				"action_id": actionID,
				"block_id":  blockID,
				"type":      "button",
				"text":      map[string]interface{}{"type": "plain_text", "text": "Click me"},
				"value":     "button_value",
			},
		},
		"response_url": "https://hooks.slack.com/actions/T123456/123456/abcdef",
		"trigger_id":   "123456.123456.abcdef",
	}

	body, _ := json.Marshal(action)
	return body
}

func TestAppActionRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route action by action_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler
		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.True(t, handlerCalled, "Action handler should have been called")
	})

	t.Run("should route action by block_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler
		blockID := "block_1"
		app.Action(bolt.ActionConstraints{
			BlockID: &blockID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.True(t, handlerCalled, "Action handler should have been called")
	})

	t.Run("should route action by callback_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler
		callbackID := "button_callback"
		app.Action(bolt.ActionConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createButtonActionBody(),
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

		assert.True(t, handlerCalled, "Action handler should have been called")
	})

	t.Run("should not route action if constraints don't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler with different action_id
		actionID := "different_button"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.False(t, handlerCalled, "Action handler should not have been called")
	})

	t.Run("should pass correct action data to handler", func(t *testing.T) {
		var receivedAction interface{}
		var receivedBody interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler that captures action data
		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedAction = args.Action
			receivedBody = args.Body
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Ack, "Ack function should be present")
			assert.NotNil(t, args.Respond, "Respond function should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.NotNil(t, receivedAction, "Action data should have been passed to handler")
		assert.NotNil(t, receivedBody, "Body data should have been passed to handler")

		// Verify action structure
		actionMap, ok := ExtractRawActionData(receivedAction.(types.SlackAction))
		require.True(t, ok, "Action should be extractable as map")
		assert.Equal(t, "button_1", actionMap["action_id"], "Action ID should be correct")
		assert.Equal(t, "block_1", actionMap["block_id"], "Block ID should be correct")
	})

	t.Run("should handle multiple constraints", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler with multiple constraints
		actionID := "button_1"
		blockID := "block_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
			BlockID:  &blockID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches both constraints
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.True(t, handlerCalled, "Action handler should have been called when all constraints match")
	})

	t.Run("should not match if one constraint fails", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register action handler with multiple constraints
		actionID := "button_1"
		blockID := "different_block"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
			BlockID:  &blockID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches action_id but not block_id
		event := types.ReceiverEvent{
			Body: createBlockActionBody("button_1", "block_1"),
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

		assert.False(t, handlerCalled, "Action handler should not have been called when constraints don't all match")
	})
}
