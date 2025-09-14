package test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOptionsRoutingComprehensive implements the missing tests from routing-options.spec.ts
func TestOptionsRoutingComprehensive(t *testing.T) {
	t.Parallel()
	t.Run("should route a block suggestion event to a handler registered with options(string) that matches the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackOptionsMiddlewareArgs
		handlerCalled := false

		// Register handler with string action ID
		app.OptionsString("my_id", func(args bolt.SlackOptionsMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Return some options
			response := &bolt.OptionsResponse{
				Options: []bolt.Option{
					*slack.NewOptionBlockObject("value1", slack.NewTextBlockObject("plain_text", "Option 1", false, false), nil),
					*slack.NewOptionBlockObject("value2", slack.NewTextBlockObject("plain_text", "Option 2", false, false), nil),
				},
			}
			return args.Ack(response)
		})

		// Create options event with matching action ID
		optionsBody := createBlockSuggestionBodyComprehensive("my_id", "block123")
		event := types.ReceiverEvent{
			Body: optionsBody,
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
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.Options, "Options should be available")

		// Verify options data
		if bodyMap, ok := receivedArgs.Body.(map[string]interface{}); ok {
			assert.Equal(t, "my_id", bodyMap["action_id"], "Action ID should match")
		}
	})

	t.Run("should route a block suggestion event to a handler registered with options(RegExp) that matches the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackOptionsMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp pattern (matches "my_*")
		actionPattern := regexp.MustCompile(`my_.*`)
		app.OptionsPattern(actionPattern, func(args bolt.SlackOptionsMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Return some options
			response := &bolt.OptionsResponse{
				Options: []bolt.Option{
					*slack.NewOptionBlockObject("value1", slack.NewTextBlockObject("plain_text", "Option 1", false, false), nil),
				},
			}
			return args.Ack(response)
		})

		// Create options event with matching action ID ("my_action" matches "my_.*")
		optionsBody := createBlockSuggestionBodyComprehensive("my_action", "block123")
		event := types.ReceiverEvent{
			Body: optionsBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for RegExp matching action ID")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.Options, "Options should be available")
	})

	t.Run("should route a block suggestion event to a handler registered with options({block_id}) that matches the block ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackOptionsMiddlewareArgs
		handlerCalled := false

		// Register handler with block ID constraint
		blockID := "my_id"
		app.Options(bolt.OptionsConstraints{
			BlockID: blockID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Return some options
			response := &bolt.OptionsResponse{
				Options: []bolt.Option{
					*slack.NewOptionBlockObject("value1", slack.NewTextBlockObject("plain_text", "Option 1", false, false), nil),
				},
			}
			return args.Ack(response)
		})

		// Create options event with matching block ID
		optionsBody := createBlockSuggestionBodyComprehensive("action123", "my_id")
		event := types.ReceiverEvent{
			Body: optionsBody,
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

		assert.True(t, handlerCalled, "Handler should have been called for block ID matching")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.Options, "Options should be available")

		// Verify block ID
		if bodyMap, ok := receivedArgs.Body.(map[string]interface{}); ok {
			assert.Equal(t, "my_id", bodyMap["block_id"], "Block ID should match")
		}
	})

	t.Run("should not execute handler if no routing found", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register handler for different action ID
		app.OptionsString("different_id", func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return args.Ack(&bolt.OptionsResponse{})
		})

		// Send options request that doesn't match
		optionsBody := createBlockSuggestionBodyComprehensive("my_id", "block123")
		event := types.ReceiverEvent{
			Body: optionsBody,
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching options")
	})

	t.Run("should route block suggestion event to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs types.SlackOptionsMiddlewareArgs
		handlerCalled := false
		ackCalled := false

		// Register handler that will acknowledge
		app.OptionsString("my_action", func(args types.SlackOptionsMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Acknowledge with options response
			response := &types.OptionsResponse{
				Options: []types.Option{
					{
						Text:  &types.TextObject{Type: "plain_text", Text: "Option 1"},
						Value: "value1",
					},
					{
						Text:  &types.TextObject{Type: "plain_text", Text: "Option 2"},
						Value: "value2",
					},
				},
			}

			err := args.Ack(response)
			if err == nil {
				ackCalled = true
			}
			return err
		})

		// Create options event
		optionsBody := createBlockSuggestionBodyComprehensive("my_action", "block123")
		event := types.ReceiverEvent{
			Body: optionsBody,
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
		assert.NotNil(t, receivedArgs.Options, "Options should be available")

		// Verify that the handler processed the request properly
		if bodyMap, ok := receivedArgs.Body.(map[string]interface{}); ok {
			assert.Equal(t, "my_action", bodyMap["action_id"], "Action ID should match")
			assert.Equal(t, "block_suggestion", bodyMap["type"], "Type should be block_suggestion")
		}
	})
}

// Helper function for creating block suggestion event bodies
func createBlockSuggestionBodyComprehensive(actionID, blockID string) []byte {
	optionsBody := map[string]interface{}{
		"type":      "block_suggestion",
		"token":     "test_token",
		"action_id": actionID,
		"block_id":  blockID,
		"value":     "test",
		"team": map[string]interface{}{
			"id":     "T123456",
			"domain": "test-team",
		},
		"user": map[string]interface{}{
			"id":   "U123456",
			"name": "testuser",
		},
		"api_app_id": "A123456",
		"container": map[string]interface{}{
			"type":    "view",
			"view_id": "V123456789",
		},
	}

	bodyBytes, _ := json.Marshal(optionsBody)
	return bodyBytes
}
