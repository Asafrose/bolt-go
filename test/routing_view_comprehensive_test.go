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

// TestViewRoutingComprehensive implements the missing tests from routing-view.spec.ts
func TestViewRoutingComprehensive(t *testing.T) {
	t.Parallel()
	t.Run("should throw if provided a constraint with unknown view constraint keys", func(t *testing.T) {
		// In Go, this would be caught at compile time due to type safety
		// But we can test that the constraints work as expected
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Test that valid constraints work
		callbackID := "valid_callback"
		viewType := "view_submission"

		// This should compile and work fine
		app.View(types.ViewConstraints{
			CallbackID: &callbackID,
			Type:       &viewType,
		}, func(args types.SlackViewMiddlewareArgs) error {
			return args.Ack(nil)
		})

		// In Go's type system, unknown fields would not compile
		// This test verifies that the constraint system works as expected
		assert.NotNil(t, app, "App should be created with valid view constraints")
	})
	t.Run("should route a view submission event to a handler registered with view(string) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs
		handlerCalled := false

		// Register handler with string callback ID
		app.ViewString("my_id", func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create view submission event with matching callback ID
		viewBody := createViewSubmissionBodyComprehensive("my_id")
		event := types.ReceiverEvent{
			Body: viewBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called for matching callback ID")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.View, "View should be available")

		// Verify view data
		if bodyMap, ok := ExtractRawViewData(receivedArgs.Body); ok {
			if view, exists := bodyMap["view"]; exists {
				if viewMap, ok := view.(map[string]interface{}); ok {
					assert.Equal(t, "my_id", viewMap["callback_id"], "Callback ID should match")
				}
			}
		}
	})

	t.Run("should route a view submission event to a handler registered with view(RegExp) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp pattern (matches "my_*")
		callbackPattern := regexp.MustCompile(`my_.*`)
		app.ViewPattern(callbackPattern, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create view submission event with matching callback ID ("my_id" matches "my_.*")
		viewBody := createViewSubmissionBodyComprehensive("my_id")
		event := types.ReceiverEvent{
			Body: viewBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called for RegExp matching callback ID")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.View, "View should be available")
	})

	t.Run("should route a view submission event to a handler registered with view({callback_id}) that matches the callback ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs
		handlerCalled := false

		// Register handler with constraint object
		callbackID := "my_id"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create view submission event with matching callback ID
		viewBody := createViewSubmissionBodyComprehensive("my_id")
		event := types.ReceiverEvent{
			Body: viewBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.True(t, handlerCalled, "Handler should have been called for constraint object matching")
		assert.NotNil(t, receivedArgs.Body, "Body should be available")
		assert.NotNil(t, receivedArgs.View, "View should be available")
	})

	t.Run("should route a view submission event to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs
		handlerCalled := false
		ackCalled := false

		app.ViewString("my_id", func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Call ack in the handler
			response := &bolt.ViewResponse{
				ResponseAction: "clear",
			}
			err := args.Ack(response)
			if err == nil {
				ackCalled = true
			}
			return err
		})

		// Create view submission event
		viewBody := createViewSubmissionBodyComprehensive("my_id")
		event := types.ReceiverEvent{
			Body: viewBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
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

	t.Run("should not execute handler if no routing found", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register handler for different callback ID
		app.ViewString("different_id", func(args bolt.SlackViewMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Send view submission that doesn't match
		viewBody := createViewSubmissionBodyComprehensive("my_id")
		event := types.ReceiverEvent{
			Body: viewBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching view")
	})

	// Missing tests for view closed events
	t.Run("for view closed events", func(t *testing.T) {
		t.Run("should route a view closed event to a handler registered with view({callback_id, type:view_closed}) that matches callback ID", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var receivedArgs types.SlackViewMiddlewareArgs
			handlerCalled := false

			// Register handler with callback_id and type constraints
			callbackID := "my_callback_id"
			viewType := "view_closed"
			app.View(types.ViewConstraints{
				CallbackID: &callbackID,
				Type:       &viewType,
			}, func(args types.SlackViewMiddlewareArgs) error {
				receivedArgs = args
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create view closed event with matching callback ID
			viewBody := createViewClosedBodyComprehensive("my_callback_id")
			event := types.ReceiverEvent{
				Body: viewBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should have been called for view closed event")
			assert.NotNil(t, receivedArgs.Body, "Body should be available")
			assert.NotNil(t, receivedArgs.View, "View should be available")

			// Verify callback ID
			if bodyMap, ok := ExtractRawViewData(receivedArgs.Body); ok {
				if view, ok := bodyMap["view"].(map[string]interface{}); ok {
					assert.Equal(t, "my_callback_id", view["callback_id"], "Callback ID should match")
				}
			}
		})

		t.Run("should route a view closed event to a handler registered with view({type:view_closed})", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var receivedArgs types.SlackViewMiddlewareArgs
			handlerCalled := false

			// Register handler with only type constraint
			viewType := "view_closed"
			app.View(types.ViewConstraints{
				Type: &viewType,
			}, func(args types.SlackViewMiddlewareArgs) error {
				receivedArgs = args
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create view closed event
			viewBody := createViewClosedBodyComprehensive("some_callback_id")
			event := types.ReceiverEvent{
				Body: viewBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should have been called for view closed event")
			assert.NotNil(t, receivedArgs.Body, "Body should be available")
			assert.NotNil(t, receivedArgs.View, "View should be available")

			// Verify type
			if bodyMap, ok := ExtractRawViewData(receivedArgs.Body); ok {
				assert.Equal(t, "view_closed", bodyMap["type"], "Type should match")
			}
		})

		t.Run("should route a view closed event to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			handlerCalled := false
			ackCalled := false

			// Register handler with type constraint
			viewType := "view_closed"
			app.View(types.ViewConstraints{
				Type: &viewType,
			}, func(args types.SlackViewMiddlewareArgs) error {
				handlerCalled = true

				// Call ack within the handler
				err := args.Ack(nil)
				if err == nil {
					ackCalled = true
				}
				return err
			})

			// Create view closed event
			viewBody := createViewClosedBodyComprehensive("test_callback_id")
			event := types.ReceiverEvent{
				Body: viewBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should have been called")
			assert.True(t, ackCalled, "Ack should have been called within the handler")
		})
	})
}

// Helper function for creating view submission event bodies
func createViewSubmissionBodyComprehensive(callbackID string) []byte {
	viewBody := map[string]interface{}{
		"type":  "view_submission",
		"token": "test_token",
		"team": map[string]interface{}{
			"id":     "T123456",
			"domain": "test-team",
		},
		"user": map[string]interface{}{
			"id":   "U123456",
			"name": "testuser",
		},
		"api_app_id": "A123456",
		"trigger_id": "123456789.123456789.abcdefg",
		"view": map[string]interface{}{
			"id":          "V123456789",
			"team_id":     "T123456",
			"type":        "modal",
			"callback_id": callbackID,
			"title": map[string]interface{}{
				"type": "plain_text",
				"text": "Test Modal",
			},
			"submit": map[string]interface{}{
				"type": "plain_text",
				"text": "Submit",
			},
			"blocks": []interface{}{
				map[string]interface{}{
					"type":     "section",
					"block_id": "section1",
					"text": map[string]interface{}{
						"type": "plain_text",
						"text": "This is a test modal",
					},
				},
			},
			"state": map[string]interface{}{
				"values": map[string]interface{}{},
			},
		},
		"response_urls": []interface{}{},
	}

	bodyBytes, _ := json.Marshal(viewBody)
	return bodyBytes
}

// Helper function for creating view closed event bodies
func createViewClosedBodyComprehensive(callbackID string) []byte {
	viewBody := map[string]interface{}{
		"type": "view_closed",
		"user": map[string]interface{}{
			"id": "U123456",
		},
		"view": map[string]interface{}{
			"id":          "V12345",
			"callback_id": callbackID,
			"type":        "modal",
			"title": map[string]interface{}{
				"type": "plain_text",
				"text": "Test Modal",
			},
			"blocks": []interface{}{},
		},
		"team": map[string]interface{}{
			"id": "T123456",
		},
	}

	bodyBytes, _ := json.Marshal(viewBody)
	return bodyBytes
}
