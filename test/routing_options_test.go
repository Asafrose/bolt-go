package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers for options routing
func createOptionsRequestBody(actionID, blockID string) []byte {
	optionsRequest := map[string]interface{}{
		"type":       "block_suggestion",
		"token":      "verification-token",
		"team":       map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":       map[string]interface{}{"id": "U123456", "name": "testuser"},
		"channel":    map[string]interface{}{"id": "C123456", "name": "general"},
		"api_app_id": "A123456",
		"action_id":  actionID,
		"block_id":   blockID,
		"value":      "test",
		"view": map[string]interface{}{
			"id":          "V123456",
			"team_id":     "T123456",
			"type":        "modal",
			"callback_id": "test_modal",
			"title":       map[string]interface{}{"type": "plain_text", "text": "Test Modal"},
		},
	}

	body, _ := json.Marshal(optionsRequest)
	return body
}

func createSelectOptionsRequestBody(actionID string) []byte {
	optionsRequest := map[string]interface{}{
		"type":        "interactive_message",
		"token":       "verification-token",
		"team":        map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":        map[string]interface{}{"id": "U123456", "name": "testuser"},
		"channel":     map[string]interface{}{"id": "C123456", "name": "general"},
		"api_app_id":  "A123456",
		"action_id":   actionID,
		"value":       "te",
		"callback_id": "legacy_callback",
		"name":        "legacy_select", // Required field for options request identification
	}

	body, _ := json.Marshal(optionsRequest)
	return body
}

func TestAppOptionsRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route options request by action_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler
		actionID := "select_1"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.True(t, handlerCalled, "Options handler should have been called")
	})

	t.Run("should route options request by block_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler
		blockID := "block_1"
		app.Options(bolt.OptionsConstraints{
			BlockID: blockID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.True(t, handlerCalled, "Options handler should have been called")
	})

	t.Run("should not route options request if action_id doesn't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler with different action_id
		actionID := "different_select"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.False(t, handlerCalled, "Options handler should not have been called")
	})

	t.Run("should pass correct options data to handler", func(t *testing.T) {
		var receivedOptions bolt.OptionsRequest
		var receivedBody interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler that captures options data
		actionID := "select_1"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			receivedOptions = args.Options
			receivedBody = args.Body
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Ack, "Ack function should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.NotNil(t, receivedOptions, "Options data should have been passed to handler")
		assert.NotNil(t, receivedBody, "Body data should have been passed to handler")

		// Verify options structure
		assert.Equal(t, "select_1", receivedOptions.ActionID, "Action ID should be correct")
		assert.Equal(t, "block_1", receivedOptions.BlockID, "Block ID should be correct")
		assert.Equal(t, "test", receivedOptions.Value, "Value should be correct")
	})

	t.Run("should handle multiple constraints", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler with multiple constraints
		actionID := "select_1"
		blockID := "block_1"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
			BlockID:  blockID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches both constraints
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.True(t, handlerCalled, "Options handler should have been called when all constraints match")
	})

	t.Run("should not match if one constraint fails", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler with multiple constraints
		actionID := "select_1"
		blockID := "different_block"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
			BlockID:  blockID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches action_id but not block_id
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
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

		assert.False(t, handlerCalled, "Options handler should not have been called when constraints don't all match")
	})

	t.Run("should handle legacy select menu options", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler
		actionID := "legacy_select"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event for legacy select menu
		event := types.ReceiverEvent{
			Body: createSelectOptionsRequestBody("legacy_select"),
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

		assert.True(t, handlerCalled, "Options handler should handle legacy select menus")
	})

	t.Run("should handle options response with ack", func(t *testing.T) {
		var ackResponse interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register options handler that returns options
		actionID := "select_1"
		app.Options(bolt.OptionsConstraints{
			ActionID: actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			// Simulate returning options using slack SDK constructors
			options := &bolt.OptionsResponse{
				Options: []bolt.Option{
					*slack.NewOptionBlockObject("value1", slack.NewTextBlockObject("plain_text", "Option 1", false, false), nil),
					*slack.NewOptionBlockObject("value2", slack.NewTextBlockObject("plain_text", "Option 2", false, false), nil),
				},
			}
			return args.Ack(options)
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createOptionsRequestBody("select_1", "block_1"),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				ackResponse = response
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.NotNil(t, ackResponse, "Options response should have been passed to ack")

		// Verify response structure
		if optionsResp, ok := ackResponse.(*bolt.OptionsResponse); ok {
			assert.Len(t, optionsResp.Options, 2, "Should have 2 options")
			assert.Equal(t, "Option 1", optionsResp.Options[0].Text.Text, "First option text should be correct")
			assert.Equal(t, "value1", optionsResp.Options[0].Value, "First option value should be correct")
		}
	})
}
