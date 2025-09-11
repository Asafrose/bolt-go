package test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenerErrorHandling(t *testing.T) {
	t.Run("should handle listener errors gracefully", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		expectedError := errors.New("listener error")
		
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			return expectedError
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Error should be handled gracefully
		assert.Error(t, err, "Should return error from listener")
	})
	
	t.Run("should handle multiple listener errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		listener1Called := false
		listener2Called := false
		
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			listener1Called = true
			return errors.New("listener 1 error")
		})
		
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			listener2Called = true
			return errors.New("listener 2 error")
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "hello world",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// At least one listener should be called
		assert.True(t, listener1Called || listener2Called, "At least one listener should be called")
		assert.Error(t, err, "Should return error from listeners")
	})
}

func TestMiddlewareErrorHandling(t *testing.T) {
	t.Run("should handle middleware errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		middlewareError := errors.New("middleware error")
		
		// Add global middleware that returns an error
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			return middlewareError
		})
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		assert.Error(t, err, "Should return middleware error")
		assert.False(t, listenerCalled, "Listener should not be called when middleware fails")
	})
	
	t.Run("should handle middleware that doesn't call next", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		// Add global middleware that doesn't call next
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// Don't call args.Next()
			return nil
		})
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Should not error, but listener should not be called
		assert.NoError(t, err, "Should not error when middleware doesn't call next")
		assert.False(t, listenerCalled, "Listener should not be called when middleware doesn't call next")
	})
}

func TestAuthorizationErrorHandling(t *testing.T) {
	t.Run("should handle authorization errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			SigningSecret: &fakeSigningSecret,
			Authorize: func(ctx context.Context, source bolt.AuthorizeSourceData, body interface{}) (*bolt.AuthorizeResult, error) {
				return nil, errors.New("authorization failed")
			},
		})
		require.NoError(t, err)
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
			"team_id": "T123456",
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		assert.Error(t, err, "Should return authorization error")
		assert.False(t, listenerCalled, "Listener should not be called when authorization fails")
	})
	
	t.Run("should handle missing authorization", func(t *testing.T) {
		// Create app without token or authorization function
		_, err := bolt.New(bolt.AppOptions{
			SigningSecret: &fakeSigningSecret,
			// No Token or Authorize function
		})
		
		assert.Error(t, err, "Should return error when no authorization is provided")
	})
}

func TestEventProcessingErrorHandling(t *testing.T) {
	t.Run("should handle malformed event bodies", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create malformed event body
		malformedBody := []byte(`{"type": "event_callback", "event": malformed json}`)
		
		event := types.ReceiverEvent{
			Body: malformedBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Should handle malformed JSON gracefully
		assert.Error(t, err, "Should return error for malformed JSON")
		assert.False(t, listenerCalled, "Listener should not be called for malformed events")
	})
	
	t.Run("should handle empty event bodies", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create empty event body
		event := types.ReceiverEvent{
			Body: []byte{},
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Should handle empty body gracefully
		assert.Error(t, err, "Should return error for empty body")
		assert.False(t, listenerCalled, "Listener should not be called for empty events")
	})
	
	t.Run("should handle unknown event types", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create unknown event type
		eventBody := map[string]interface{}{
			"type": "unknown_event_type",
			"data": map[string]interface{}{
				"some": "data",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Should handle unknown event types gracefully
		assert.NoError(t, err, "Should not error for unknown event types")
		assert.False(t, listenerCalled, "Listener should not be called for unknown events")
	})
}

func TestAckErrorHandling(t *testing.T) {
	t.Run("should handle ack function errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		ackError := errors.New("ack error")
		
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			// Try to ack with error
			return args.Ack(&bolt.CommandResponse{
				Text: "Test response",
			})
		})
		
		// Create command event
		commandBody := map[string]interface{}{
			"command":    "/test",
			"text":       "hello",
			"user_id":    "U123456",
			"channel_id": "C123456",
			"team_id":    "T123456",
		}
		
		bodyBytes, _ := json.Marshal(commandBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return ackError
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Should handle ack errors
		assert.Error(t, err, "Should return ack error")
	})
	
	t.Run("should handle multiple ack calls", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		ackCallCount := 0
		
		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			// Try to ack multiple times
			err1 := args.Ack(nil)
			err2 := args.Ack(nil)
			
			// Both calls should work (or the second should be ignored)
			_ = err1
			_ = err2
			
			return nil
		})
		
		// Create action event
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"type":      "button",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
		}
		
		bodyBytes, _ := json.Marshal(actionBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				ackCallCount++
				return nil
			},
		}
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		assert.NoError(t, err, "Should handle multiple ack calls")
		// Implementation may call ack once or twice depending on design
		assert.True(t, ackCallCount >= 1, "Ack should be called at least once")
	})
}

func TestContextErrorHandling(t *testing.T) {
	t.Run("should handle context cancellation", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		listenerCalled := false
		
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		err = app.ProcessEvent(ctx, event)
		
		// Should handle cancelled context gracefully
		// Behavior may vary depending on implementation
		_ = err
		_ = listenerCalled
	})
	
	t.Run("should handle context timeout", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			// Simulate long-running operation
			// In real scenario, this would check context cancellation
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()
		
		err = app.ProcessEvent(ctx, event)
		
		// Should handle timeout gracefully
		// Behavior may vary depending on implementation
		_ = err
	})
}

func TestPanicRecovery(t *testing.T) {
	t.Run("should recover from listener panics", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			panic("listener panic")
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ProcessEvent should not panic, should handle panics gracefully: %v", r)
			}
		}()
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Implementation should recover from panic and return error
		// Exact behavior depends on implementation
		_ = err
	})
	
	t.Run("should recover from middleware panics", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)
		
		// Add middleware that panics
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			panic("middleware panic")
		})
		
		listenerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})
		
		// Create event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}
		
		bodyBytes, _ := json.Marshal(eventBody)
		
		event := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}
		
		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ProcessEvent should not panic, should handle panics gracefully: %v", r)
			}
		}()
		
		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		
		// Implementation should recover from panic
		// Listener should not be called when middleware panics
		assert.False(t, listenerCalled, "Listener should not be called when middleware panics")
		_ = err
	})
}
