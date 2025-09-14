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

// TestCustomFunctionRouting implements the missing tests from routing-function.spec.ts
func TestCustomFunctionRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a function executed event to a handler registered with function(string) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCustomFunctionMiddlewareArgs
		handlerCalled := false

		// Register function handler with callback ID
		app.Function("my_id", func(args bolt.SlackCustomFunctionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return args.Next()
		})

		// Create function_executed event with matching callback ID
		functionBody := createFunctionExecutedEventBody("my_id", map[string]interface{}{"test": true})
		event := types.ReceiverEvent{
			Body: functionBody,
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
		assert.NotNil(t, receivedArgs.Event, "Event should be available")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.Payload, "Payload should be available")
	})

	t.Run("should route a function executed event to a handler with the proper arguments", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCustomFunctionMiddlewareArgs
		handlerCalled := false
		testInputs := map[string]interface{}{"test": true}

		// Register function handler with callback ID
		app.Function("my_id", func(args bolt.SlackCustomFunctionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			_ = receivedArgs // Use the variable to avoid "declared and not used" error

			// Verify inputs are available in the event
			if eventMap, ok := args.Event.(map[string]interface{}); ok {
				if function, exists := eventMap["function"]; exists {
					if functionMap, ok := function.(map[string]interface{}); ok {
						if inputs, exists := functionMap["inputs"]; exists {
							if inputsMap, ok := inputs.(map[string]interface{}); ok {
								assert.Equal(t, testInputs, inputsMap, "Inputs should match")
							}
						}
					}
				}
			}

			// Verify utility functions are available (would be populated in real implementation)
			assert.NotNil(t, args.Complete, "Complete function should be available")
			assert.NotNil(t, args.Fail, "Fail function should be available")
			assert.NotNil(t, args.Client, "Client should be available")

			return args.Next()
		})

		// Create function_executed event with inputs
		functionBody := createFunctionExecutedEventBody("my_id", testInputs)
		event := types.ReceiverEvent{
			Body: functionBody,
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
	})

	t.Run("should route a function executed event to a handler and auto ack by default", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false
		ackCalled := false

		// Register function handler (auto-acknowledge is default)
		app.Function("my_id", func(args bolt.SlackCustomFunctionMiddlewareArgs) error {
			handlerCalled = true
			return args.Next()
		})

		// Create function_executed event
		functionBody := createFunctionExecutedEventBody("my_id", map[string]interface{}{})
		event := types.ReceiverEvent{
			Body: functionBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				ackCalled = true
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.True(t, ackCalled, "Event should have been auto-acknowledged")
	})

	t.Run("should route a function executed event to a handler and NOT auto ack if autoAcknowledge is false", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false
		ackCalled := false

		options := bolt.CustomFunctionOptions{AutoAcknowledge: false}

		// Register function handler with auto-acknowledge disabled
		app.Function("my_id", options, func(args bolt.SlackCustomFunctionMiddlewareArgs) error {
			handlerCalled = true
			return args.Next()
		})

		// Create function_executed event
		functionBody := createFunctionExecutedEventBody("my_id", map[string]interface{}{})
		event := types.ReceiverEvent{
			Body: functionBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				ackCalled = true
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.False(t, ackCalled, "Event should NOT have been auto-acknowledged")
	})

	t.Run("should not execute handler if callback ID doesn't match", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register function handler for different callback ID
		app.Function("different_id", func(args bolt.SlackCustomFunctionMiddlewareArgs) error {
			handlerCalled = true
			return args.Next()
		})

		// Create function_executed event with non-matching callback ID
		functionBody := createFunctionExecutedEventBody("my_id", map[string]interface{}{})
		event := types.ReceiverEvent{
			Body: functionBody,
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching callback ID")
	})
}

// Helper function for creating function_executed event bodies
func createFunctionExecutedEventBody(callbackID string, inputs map[string]interface{}) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type": "function_executed",
			"function": map[string]interface{}{
				"id":          "Fn123456",
				"callback_id": callbackID,
				"title":       "Test Function",
				"description": "A test custom function",
				"type":        "app",
				"inputs":      inputs,
				"outputs":     map[string]interface{}{},
			},
			"inputs":                inputs,
			"function_execution_id": "Fx123456789",
			"workflow": map[string]interface{}{
				"id": "Wf123456",
			},
			"event_ts": "1234567890.123456",
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{"U123456"},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}
