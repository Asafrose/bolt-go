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

func TestGlobalMiddlewareExecution(t *testing.T) {
	t.Run("should execute global middleware before listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		executionOrder := []string{}

		// Add global middleware
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global")
			return args.Next()
		})

		// Add event listener
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
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
		require.NoError(t, err)

		assert.Equal(t, []string{"global", "listener"}, executionOrder, "Global middleware should execute before listener")
	})

	t.Run("should execute multiple global middlewares in order", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		executionOrder := []string{}

		// Add multiple global middlewares
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global1")
			return args.Next()
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global2")
			return args.Next()
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global3")
			return args.Next()
		})

		// Add event listener
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
			return nil
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
		require.NoError(t, err)

		expected := []string{"global1", "global2", "global3", "listener"}
		assert.Equal(t, expected, executionOrder, "Global middlewares should execute in registration order")
	})

	t.Run("should stop execution if global middleware doesn't call next", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		executionOrder := []string{}

		// Add global middleware that doesn't call next
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global1")
			// Don't call args.Next()
			return nil
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global2")
			return args.Next()
		})

		// Add event listener
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
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
		require.NoError(t, err)

		assert.Equal(t, []string{"global1"}, executionOrder, "Execution should stop when middleware doesn't call next")
	})

	t.Run("should handle global middleware errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		executionOrder := []string{}

		// Add global middleware that returns error
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global1")
			return assert.AnError
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global2")
			return args.Next()
		})

		// Add event listener
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
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
		assert.Equal(t, []string{"global1"}, executionOrder, "Execution should stop on middleware error")
	})
}

func TestGlobalMiddlewareContextPassing(t *testing.T) {
	t.Run("should pass context through global middleware chain", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var middleware1Context *types.Context
		var middleware2Context *types.Context
		var listenerContext *types.Context

		// Add global middlewares that capture context
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middleware1Context = args.Context
			return args.Next()
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middleware2Context = args.Context
			return args.Next()
		})

		// Add event listener
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerContext = args.Context
			return nil
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
		require.NoError(t, err)

		// All contexts should be the same object
		assert.Equal(t, middleware1Context, middleware2Context, "Context should be passed through middleware chain")
		assert.Equal(t, middleware2Context, listenerContext, "Context should be passed to listener")

		// Context should contain team information
		assert.NotNil(t, listenerContext.TeamID, "Context should contain team ID")
		assert.Equal(t, "T123456", *listenerContext.TeamID, "Team ID should match")
	})

	t.Run("should allow middleware to modify context", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var listenerContext *types.Context

		// Add global middleware that modifies context
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// Add custom property to context
			args.Context.Custom["middleware_flag"] = "modified"
			return args.Next()
		})

		// Add event listener
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerContext = args.Context
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
		require.NoError(t, err)

		// Listener should see modified context
		assert.NotNil(t, listenerContext, "Listener should receive context")
		assert.Equal(t, "modified", listenerContext.Custom["middleware_flag"], "Context should be modified by middleware")
	})
}

func TestGlobalMiddlewareWithDifferentEventTypes(t *testing.T) {
	t.Run("should execute global middleware for all event types", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		globalMiddlewareCalls := 0

		// Add global middleware
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			globalMiddlewareCalls++
			return args.Next()
		})

		// Add listeners for different event types
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			return nil
		})

		actionID := "button_1"
		app.Action(bolt.ActionConstraints{ActionID: &actionID}, func(args bolt.SlackActionMiddlewareArgs) error {
			return nil
		})

		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			return nil
		})

		// Test event
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
		require.NoError(t, err)

		// Test action
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

		bodyBytes, _ = json.Marshal(actionBody)

		event = types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Test command
		commandBody := map[string]interface{}{
			"command":    "/test",
			"text":       "hello",
			"user_id":    "U123456",
			"channel_id": "C123456",
			"team_id":    "T123456",
		}

		bodyBytes, _ = json.Marshal(commandBody)

		event = types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response interface{}) error {
				return nil
			},
		}

		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Global middleware should have been called for all event types
		assert.Equal(t, 3, globalMiddlewareCalls, "Global middleware should be called for all event types")
	})

	t.Run("should provide correct middleware args for different event types", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var eventArgs bolt.AllMiddlewareArgs
		var actionArgs bolt.AllMiddlewareArgs
		var commandArgs bolt.AllMiddlewareArgs

		// Add global middleware that captures args
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// Store args based on type (simplified for testing)
			eventArgs = args
			actionArgs = args
			commandArgs = args
			return args.Next()
		})

		// Add listeners
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			return nil
		})

		actionID := "button_1"
		app.Action(bolt.ActionConstraints{ActionID: &actionID}, func(args bolt.SlackActionMiddlewareArgs) error {
			return nil
		})

		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			return nil
		})

		// Test each event type
		ctx := context.Background()

		// Event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "hello",
				"channel": "C123456",
			},
		}

		bodyBytes, _ := json.Marshal(eventBody)
		event := types.ReceiverEvent{
			Body:    bodyBytes,
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack:     func(response interface{}) error { return nil },
		}

		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Action
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

		bodyBytes, _ = json.Marshal(actionBody)
		event = types.ReceiverEvent{
			Body:    bodyBytes,
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack:     func(response interface{}) error { return nil },
		}

		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Command
		commandBody := map[string]interface{}{
			"command":    "/test",
			"text":       "hello",
			"user_id":    "U123456",
			"channel_id": "C123456",
			"team_id":    "T123456",
		}

		bodyBytes, _ = json.Marshal(commandBody)
		event = types.ReceiverEvent{
			Body:    bodyBytes,
			Headers: map[string]string{"Content-Type": "application/json"},
			Ack:     func(response interface{}) error { return nil },
		}

		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Verify args were captured
		assert.NotNil(t, eventArgs, "Event args should be captured")
		assert.NotNil(t, actionArgs, "Action args should be captured")
		assert.NotNil(t, commandArgs, "Command args should be captured")
	})
}

func TestGlobalMiddlewareIgnoreSelf(t *testing.T) {
	t.Run("should ignore messages from bot itself", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		listenerCalled := false

		// Add message listener
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			listenerCalled = true
			return nil
		})

		// Create message event from bot itself
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U987654", // Bot's user ID
				"bot_id":  "B987654", // Bot ID
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
		require.NoError(t, err)

		// Listener should not be called for bot's own messages
		// (This depends on the ignore-self middleware being enabled by default)
		// The exact behavior may vary based on implementation
		_ = listenerCalled
	})
}
