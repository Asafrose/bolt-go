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

// Test helpers for view routing
func createViewSubmissionBody(callbackID string) []byte {
	viewSubmission := map[string]interface{}{
		"type":       "view_submission",
		"token":      "verification-token",
		"team":       map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":       map[string]interface{}{"id": "U123456", "name": "testuser"},
		"api_app_id": "A123456",
		"trigger_id": "123456.123456.abcdef",
		"view": map[string]interface{}{
			"id":               "V123456",
			"team_id":          "T123456",
			"type":             "modal",
			"callback_id":      callbackID,
			"title":            map[string]interface{}{"type": "plain_text", "text": "Test Modal"},
			"submit":           map[string]interface{}{"type": "plain_text", "text": "Submit"},
			"close":            map[string]interface{}{"type": "plain_text", "text": "Cancel"},
			"private_metadata": "",
			"state": map[string]interface{}{
				"values": map[string]interface{}{
					"block_1": map[string]interface{}{
						"input_1": map[string]interface{}{
							"type":  "plain_text_input",
							"value": "test value",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(viewSubmission)
	return body
}

func createViewClosedBody(callbackID string) []byte {
	viewClosed := map[string]interface{}{
		"type":       "view_closed",
		"token":      "verification-token",
		"team":       map[string]interface{}{"id": "T123456", "domain": "testteam"},
		"user":       map[string]interface{}{"id": "U123456", "name": "testuser"},
		"api_app_id": "A123456",
		"trigger_id": "123456.123456.abcdef",
		"view": map[string]interface{}{
			"id":               "V123456",
			"team_id":          "T123456",
			"type":             "modal",
			"callback_id":      callbackID,
			"title":            map[string]interface{}{"type": "plain_text", "text": "Test Modal"},
			"submit":           map[string]interface{}{"type": "plain_text", "text": "Submit"},
			"close":            map[string]interface{}{"type": "plain_text", "text": "Cancel"},
			"private_metadata": "",
		},
		"is_cleared": false,
	}

	body, _ := json.Marshal(viewClosed)
	return body
}

func TestAppViewRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route view submission by callback_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler
		callbackID := "test_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.True(t, handlerCalled, "View handler should have been called")
	})

	t.Run("should route view closed by callback_id", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler
		callbackID := "test_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewClosedBody("test_modal"),
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

		assert.True(t, handlerCalled, "View closed handler should have been called")
	})

	t.Run("should route view by type", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler by type
		viewType := "view_submission"
		app.View(bolt.ViewConstraints{
			Type: &viewType,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("any_callback"),
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

		assert.True(t, handlerCalled, "View handler should have been called by type")
	})

	t.Run("should not route view if callback_id doesn't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler with different callback_id
		callbackID := "different_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.False(t, handlerCalled, "View handler should not have been called")
	})

	t.Run("should pass correct view data to handler", func(t *testing.T) {
		var receivedView interface{}
		var receivedBody interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler that captures view data
		callbackID := "test_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedView = args.View
			receivedBody = args.Body
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Ack, "Ack function should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.NotNil(t, receivedView, "View data should have been passed to handler")
		assert.NotNil(t, receivedBody, "Body data should have been passed to handler")

		// Verify view structure (simplified since we're using interface{})
		assert.NotNil(t, receivedView, "View should be present")
	})

	t.Run("should handle multiple constraints", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler with multiple constraints
		callbackID := "test_modal"
		viewType := "view_submission"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
			Type:       &viewType,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event that matches both constraints
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.True(t, handlerCalled, "View handler should have been called when all constraints match")
	})

	t.Run("should not match if type constraint fails", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler with wrong type constraint
		callbackID := "test_modal"
		viewType := "view_closed"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
			Type:       &viewType,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event for view_submission
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.False(t, handlerCalled, "View handler should not have been called when type constraint doesn't match")
	})

	t.Run("should handle view submission with form data", func(t *testing.T) {
		var formValues map[string]map[string]interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register view handler that extracts form data
		callbackID := "test_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			// Extract form values from the strongly typed view
			formValues = args.View.Values
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createViewSubmissionBody("test_modal"),
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

		assert.NotNil(t, formValues, "Form values should have been extracted")

		// Verify form structure
		if block1, exists := formValues["block_1"]; exists {
			if input1, exists := block1["input_1"]; exists {
				if input1Map, ok := input1.(map[string]interface{}); ok {
					assert.Equal(t, "test value", input1Map["value"], "Form input value should be correct")
				}
			}
		}
	})
}
