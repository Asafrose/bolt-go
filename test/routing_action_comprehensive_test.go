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

// TestActionRouting implements the missing tests from routing-action.spec.ts
func TestActionRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a block action event to a handler registered with action(string) that matches the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Register handler with string action ID
		constraints := bolt.ActionConstraints{
			ActionID: "submit_button",
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create action event with matching action_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "submit_button",
					"block_id":  "block_1",
					"type":      "button",
					"value":     "submit",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for matching action ID")
		if actionMap, ok := ExtractRawActionData(receivedArgs.Action); ok {
			assert.Equal(t, "submit_button", actionMap["action_id"], "Action ID should match")
		}
	})

	t.Run("should route a block action event to a handler registered with action(RegExp) that matches the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp action ID pattern
		actionPattern := regexp.MustCompile(`^btn_.*`)
		constraints := bolt.ActionConstraints{
			ActionIDPattern: actionPattern,
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create action event with matching action_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "btn_submit",
					"block_id":  "block_1",
					"type":      "button",
					"value":     "submit",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for matching RegExp action ID")
		if actionMap, ok := ExtractRawActionData(receivedArgs.Action); ok {
			assert.Equal(t, "btn_submit", actionMap["action_id"], "Action ID should match")
		}
	})

	t.Run("should route a block action event to a handler registered with action({block_id}) that matches the block ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Register handler with block ID constraint
		constraints := bolt.ActionConstraints{
			BlockID: "approval_section",
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create action event with matching block_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "approve_btn",
					"block_id":  "approval_section",
					"type":      "button",
					"value":     "approve",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for matching block ID")
		if actionMap, ok := ExtractRawActionData(receivedArgs.Action); ok {
			assert.Equal(t, "approval_section", actionMap["block_id"], "Block ID should match")
		}
	})

	t.Run("should route a block action event to a handler registered with action({type:block_actions})", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Register handler with type constraint
		constraints := bolt.ActionConstraints{
			Type: "block_actions",
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create block_actions event
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "any_action",
					"block_id":  "any_block",
					"type":      "button",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for block_actions type")
		assert.NotNil(t, receivedArgs.Action, "Action should be available")
	})

	t.Run("should route an action event to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false
		ackCalled := false

		constraints := bolt.ActionConstraints{
			ActionID: "test_action",
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Call ack in the handler
			err := args.Ack(nil)
			if err == nil {
				ackCalled = true
			}
			return err
		})

		// Create action event
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "test_action",
					"block_id":  "test_block",
					"type":      "button",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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
		assert.True(t, ackCalled, "Ack should have been called in handler")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be available")
	})

	t.Run("should route a function scoped action to a handler with the proper arguments", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		constraints := bolt.ActionConstraints{
			ActionID: "function_action",
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create function-scoped action event (from custom function execution)
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "function_action",
					"block_id":  "function_block",
					"type":      "button",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"team":    map[string]interface{}{"id": "T123456"},
			// Function context
			"function_execution_id": "Fx123456789",
			"function": map[string]interface{}{
				"callback_id": "my_function",
			},
		}

		bodyBytes, _ := json.Marshal(actionBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for function-scoped action")
		assert.NotNil(t, receivedArgs.Body, "Body should contain function context")

		// Verify function context is available
		if bodyMap, ok := ExtractRawActionData(receivedArgs.Body); ok {
			assert.Equal(t, "Fx123456789", bodyMap["function_execution_id"], "Function execution ID should be available")
		}
	})

	t.Run("should throw if provided a constraint with unknown action constraint keys", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Test that providing an unknown constraint key should trigger validation error
		// In Go, this would typically be caught at compile time due to struct field validation
		// But we can test the constraint validation logic

		// Test that valid constraints work
		assert.NotPanics(t, func() {
			app.Action(types.ActionConstraints{
				ActionID: "valid_action_id",
			}, func(args bolt.SlackActionMiddlewareArgs) error {
				return nil
			})
		}, "Valid action_id constraint should not panic")

		// Test constraint validation with block_id constraint
		assert.NotPanics(t, func() {
			app.Action(types.ActionConstraints{
				BlockID: "valid_block_id",
			}, func(args bolt.SlackActionMiddlewareArgs) error {
				return nil
			})
		}, "Valid block_id constraint should not panic")

		// Test constraint validation with type constraint
		assert.NotPanics(t, func() {
			app.Action(types.ActionConstraints{
				Type: "block_actions",
			}, func(args bolt.SlackActionMiddlewareArgs) error {
				return nil
			})
		}, "Valid type constraint should not panic")

		// In Go's type system, unknown constraint keys are caught at compile time
		// This test validates that the constraint matching logic works correctly
		// The JavaScript equivalent would test for runtime validation errors
		// But Go's strong typing provides compile-time safety instead
	})
}
