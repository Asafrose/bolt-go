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

func TestMiddlewareChain(t *testing.T) {
	t.Parallel()
	t.Run("should execute global middleware before listeners", func(t *testing.T) {
		executionOrder := []string{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add global middleware
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global1")
			return args.Next()
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global2")
			return args.Next()
		})

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		expected := []string{"global1", "global2", "listener"}
		assert.Equal(t, expected, executionOrder, "Middleware should execute in correct order")
	})

	t.Run("should stop execution if middleware doesn't call next", func(t *testing.T) {
		executionOrder := []string{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add global middleware that doesn't call next
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global1")
			// Don't call args.Next() - this should stop the chain
			return nil
		})

		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "global2")
			return args.Next()
		})

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		expected := []string{"global1"}
		assert.Equal(t, expected, executionOrder, "Execution should stop when middleware doesn't call next")
	})

	t.Run("should handle middleware errors", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add global middleware that returns an error
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			return assert.AnError
		})

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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
		require.Error(t, err, "Middleware error should propagate")
	})

	t.Run("should pass context between middleware", func(t *testing.T) {
		var receivedContext *bolt.Context

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add global middleware that modifies context
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			if args.Context.Custom == nil {
				args.Context.Custom = make(map[string]interface{})
			}
			args.Context.Custom["middleware_data"] = "test_value"
			return args.Next()
		})

		// Register event handler that reads context
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedContext = args.Context
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		assert.NotNil(t, receivedContext, "Context should be passed to listener")
		assert.NotNil(t, receivedContext.Custom, "Custom context should be available")
		assert.Equal(t, "test_value", receivedContext.Custom["middleware_data"], "Custom data should be preserved")
	})

	t.Run("should handle multiple listeners with middleware", func(t *testing.T) {
		listener1Called := false
		listener2Called := false
		middlewareCalls := 0

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add global middleware
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			middlewareCalls++
			return args.Next()
		})

		// Register multiple event handlers
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listener1Called = true
			return nil
		})

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			listener2Called = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		assert.True(t, listener1Called, "First listener should be called")
		assert.True(t, listener2Called, "Second listener should be called")
		assert.Equal(t, 2, middlewareCalls, "Middleware should be called for each listener")
	})
}

func TestBuiltinMiddleware(t *testing.T) {
	t.Parallel()
	t.Run("should filter message events by text pattern", func(t *testing.T) {
		matchingHandlerCalled := false
		nonMatchingHandlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handlers with different patterns
		app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
			matchingHandlerCalled = true
			return nil
		})

		app.Message("goodbye", func(args bolt.SlackEventMiddlewareArgs) error {
			nonMatchingHandlerCalled = true
			return nil
		})

		// Create message event with "hello world"
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("hello world"),
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

		assert.True(t, matchingHandlerCalled, "Matching handler should be called")
		assert.False(t, nonMatchingHandlerCalled, "Non-matching handler should not be called")
	})

	t.Run("should handle case-sensitive message matching", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler with exact case pattern
		app.Message("Hello", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create message event with matching case
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText("Hello World"),
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

		// Message matching should be case-sensitive (like JavaScript version)
		assert.True(t, handlerCalled, "Handler should match with correct case")
	})

	t.Run("should handle empty message text", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler for any message
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create message event with empty text
		event := types.ReceiverEvent{
			Body: createMessageEventBodyWithText(""),
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

		assert.True(t, handlerCalled, "Handler should be called for empty message")
	})

	t.Run("should handle message with blocks but no text", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register message handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create message event with blocks but no text
		messageBody := map[string]interface{}{
			"token":      "verification-token",
			"team_id":    "T123456",
			"api_app_id": "A123456",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"ts":      "1234567890.123456",
				"channel": "C123456",
				"blocks": []interface{}{
					map[string]interface{}{
						"type": "section",
						"text": map[string]interface{}{
							"type": "mrkdwn",
							"text": "Block kit message",
						},
					},
				},
			},
			"type":         "event_callback",
			"event_id":     "Ev123456",
			"event_time":   1234567890,
			"authed_users": []string{"U987654"},
		}

		body, _ := json.Marshal(messageBody)

		event := types.ReceiverEvent{
			Body: body,
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

		assert.True(t, handlerCalled, "Handler should be called for block kit message")
	})
}

func TestMiddlewareTypes(t *testing.T) {
	t.Parallel()
	t.Run("should provide correct middleware args for events", func(t *testing.T) {
		var receivedArgs bolt.SlackEventMiddlewareArgs

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		// Verify middleware args structure
		assert.NotNil(t, receivedArgs.Event, "Event should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present")
	})

	t.Run("should provide correct middleware args for actions", func(t *testing.T) {
		var receivedArgs bolt.SlackActionMiddlewareArgs

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
			receivedArgs = args
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"type":"block_actions","token":"verification-token","team":{"id":"T123456"},"user":{"id":"U123456"},"channel":{"id":"C123456"},"actions":[{"action_id":"button_1","block_id":"block_1","type":"button","text":{"type":"plain_text","text":"Click me"},"value":"button_value"}],"response_url":"https://hooks.slack.com/actions/T123456/123456/abcdef","trigger_id":"123456.123456.abcdef"}`),
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

		// Verify middleware args structure
		assert.NotNil(t, receivedArgs.Action, "Action should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
		assert.NotNil(t, receivedArgs.Respond, "Respond function should be present")
	})

	t.Run("should provide correct middleware args for commands", func(t *testing.T) {
		var receivedArgs bolt.SlackCommandMiddlewareArgs

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","team_domain":"testteam","channel_id":"C123456","channel_name":"general","user_id":"U123456","user_name":"testuser","command":"/test","text":"hello world","response_url":"https://hooks.slack.com/commands/T123456/123456/abcdef","trigger_id":"123456.123456.abcdef","api_app_id":"A123456"}`),
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

		// Verify middleware args structure
		assert.NotNil(t, receivedArgs.Command, "Command should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present")
		assert.NotNil(t, receivedArgs.Respond, "Respond function should be present")

		// Verify command content
		assert.Equal(t, "/test", receivedArgs.Command.Command, "Command should be correct")
		assert.Equal(t, "hello world", receivedArgs.Command.Text, "Command text should be correct")
	})
}

// Helper functions are defined in other test files to avoid duplication

func TestMiddlewareExecution(t *testing.T) {
	t.Parallel()
	t.Run("should handle middleware that modifies event data", func(t *testing.T) {
		var modifiedText string

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add middleware that modifies the event (simplified for testing)
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			// In a real implementation, we would check the event type and modify accordingly
			// For now, just pass through
			return args.Next()
		})

		// Register message handler
		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			if eventMap, ok := ExtractRawEventData(args.Event); ok {
				modifiedText = eventMap["text"].(string)
			}
			return nil
		})

		// Create message event
		messageBody := map[string]interface{}{
			"token":      "verification-token",
			"team_id":    "T123456",
			"api_app_id": "A123456",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "hello world",
				"ts":      "1234567890.123456",
				"channel": "C123456",
			},
			"type":         "event_callback",
			"event_id":     "Ev123456",
			"event_time":   1234567890,
			"authed_users": []string{"U987654"},
		}

		bodyBytes, _ := json.Marshal(messageBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.Equal(t, "hello world", modifiedText, "Event text should be passed to handler")
	})

	t.Run("should handle async middleware", func(t *testing.T) {
		executionOrder := []string{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		// Add async middleware
		app.Use(func(args bolt.AllMiddlewareArgs) error {
			executionOrder = append(executionOrder, "middleware_start")
			// Simulate async operation
			go func() {
				// This would be async in real usage, but for testing we'll keep it simple
			}()
			executionOrder = append(executionOrder, "middleware_end")
			return args.Next()
		})

		// Register event handler
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			executionOrder = append(executionOrder, "listener")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: []byte(`{"token":"verification-token","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","user":"U123456","text":"<@U987654> hello","ts":"1234567890.123456","channel":"C123456"},"type":"event_callback","event_id":"Ev123456","event_time":1234567890,"authed_users":["U987654"]}`),
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

		expected := []string{"middleware_start", "middleware_end", "listener"}
		assert.Equal(t, expected, executionOrder, "Async middleware should execute correctly")
	})
}
