package test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/middleware"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuiltinMiddlewareCore tests the core builtin middleware functionality
// This implements the missing tests from builtin.spec.ts
func TestBuiltinMiddlewareCore(t *testing.T) {
	t.Parallel()
	t.Run("OnlyActions", func(t *testing.T) {
		t.Run("should only process action events", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var handlerCalled bool

			// Add OnlyActions middleware
			app.Use(middleware.OnlyActions)

			// Add a listener that should only be called for actions
			app.Action(bolt.ActionConstraints{}, func(args bolt.SlackActionMiddlewareArgs) error {
				handlerCalled = true
				return nil
			})

			// Test with action event - should process
			actionBody := createBlockActionBodyBuiltin("test_action", "test_block")
			actionEvent := types.ReceiverEvent{
				Body: actionBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error { return nil },
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, actionEvent)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should be called for action events")

			// Reset
			handlerCalled = false

			// Test with non-action event (command) - should not process
			commandBody := createCommandBody("/test", "hello")
			commandEvent := types.ReceiverEvent{
				Body: commandBody,
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Ack: func(response interface{}) error { return nil },
			}

			err = app.ProcessEvent(ctx, commandEvent)
			require.NoError(t, err)

			assert.False(t, handlerCalled, "Handler should NOT be called for non-action events")
		})
	})

	t.Run("OnlyCommands", func(t *testing.T) {
		t.Run("should only process command events", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var handlerCalled bool

			// Add OnlyCommands middleware
			app.Use(middleware.OnlyCommands)

			// Add a command listener
			app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
				handlerCalled = true
				return nil
			})

			// Test with command event - should process
			commandBody := createCommandBody("/test", "hello")
			commandEvent := types.ReceiverEvent{
				Body: commandBody,
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Ack: func(response interface{}) error { return nil },
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, commandEvent)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should be called for command events")

			// Reset
			handlerCalled = false

			// Test with non-command event (action) - should not process
			actionBody := createBlockActionBodyBuiltin("test_action", "test_block")
			actionEvent := types.ReceiverEvent{
				Body: actionBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error { return nil },
			}

			err = app.ProcessEvent(ctx, actionEvent)
			require.NoError(t, err)

			assert.False(t, handlerCalled, "Handler should NOT be called for non-command events")
		})
	})

	t.Run("OnlyEvents", func(t *testing.T) {
		t.Run("should only process event-api events", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var handlerCalled bool

			// Add OnlyEvents middleware
			app.Use(middleware.OnlyEvents)

			// Add an event listener
			app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
				handlerCalled = true
				return nil
			})

			// Test with event - should process
			eventBody := createMessageEventBodyBuiltin("U123456", "C123456", "hello world")
			eventEvent := types.ReceiverEvent{
				Body: eventBody,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response interface{}) error { return nil },
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, eventEvent)
			require.NoError(t, err)

			assert.True(t, handlerCalled, "Handler should be called for event-api events")

			// Reset
			handlerCalled = false

			// Test with non-event (command) - should not process
			commandBody := createCommandBody("/test", "hello")
			commandEvent := types.ReceiverEvent{
				Body: commandBody,
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Ack: func(response interface{}) error { return nil },
			}

			err = app.ProcessEvent(ctx, commandEvent)
			require.NoError(t, err)

			assert.False(t, handlerCalled, "Handler should NOT be called for non-event events")
		})
	})

	t.Run("IgnoreSelf", func(t *testing.T) {
		t.Run("should ignore events from the bot itself", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			middlewareCalled := false

			// Set bot user ID in context manually (normally set by authorization)
			app.Use(func(args bolt.AllMiddlewareArgs) error {
				args.Context.BotUserID = stringPtr("B123456")
				return args.Next()
			})

			app.Use(bolt.IgnoreSelf())

			app.Use(func(args bolt.AllMiddlewareArgs) error {
				middlewareCalled = true
				return args.Next()
			})

			// Create event from bot itself (should be ignored)
			eventBody := createMessageEventBodyBuiltin("B123456", "C123456", "Hello from bot")
			event := types.ReceiverEvent{
				Body: eventBody,
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

			assert.False(t, middlewareCalled, "Middleware should NOT be called for bot's own message")

			// Reset and test with different user (should proceed)
			middlewareCalled = false
			eventBody = createMessageEventBodyBuiltin("U987654", "C123456", "Hello from user")
			event.Body = eventBody

			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.True(t, middlewareCalled, "Middleware should be called for user message")
		})
	})

	t.Run("DirectMention", func(t *testing.T) {
		t.Run("should only process messages that directly mention the bot", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			middlewareCalled := false

			// Set bot user ID in context
			app.Use(func(args bolt.AllMiddlewareArgs) error {
				args.Context.BotUserID = stringPtr("B123456")
				return args.Next()
			})

			app.Use(middleware.DirectMention())

			app.Use(func(args bolt.AllMiddlewareArgs) error {
				middlewareCalled = true
				return args.Next()
			})

			// Test message with direct mention (should proceed)
			eventBody := createMessageEventBodyBuiltin("U987654", "C123456", "<@B123456> hello bot")
			event := types.ReceiverEvent{
				Body: eventBody,
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

			assert.True(t, middlewareCalled, "Middleware should be called for direct mention")

			// Reset and test without direct mention (should be ignored)
			middlewareCalled = false
			eventBody = createMessageEventBodyBuiltin("U987654", "C123456", "hello everyone")
			event.Body = eventBody

			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.False(t, middlewareCalled, "Middleware should NOT be called for non-direct mention")
		})
	})

	t.Run("MatchEventType", func(t *testing.T) {
		t.Run("should only process events matching specific type", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			middlewareCalled := false

			app.Use(middleware.MatchEventType("app_mention"))

			app.Use(func(args bolt.AllMiddlewareArgs) error {
				middlewareCalled = true
				return args.Next()
			})

			// Test app_mention event (should proceed)
			eventBody := createAppMentionEventBodyBuiltin("U987654", "C123456", "<@B123456> hello")
			event := types.ReceiverEvent{
				Body: eventBody,
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

			assert.True(t, middlewareCalled, "Middleware should be called for matching event type")

			// Reset and test with different event type (should be ignored)
			middlewareCalled = false
			eventBody = createMessageEventBodyBuiltin("U987654", "C123456", "hello")
			event.Body = eventBody

			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.False(t, middlewareCalled, "Middleware should NOT be called for non-matching event type")
		})
	})

	t.Run("MatchCommandName", func(t *testing.T) {
		t.Run("should only process commands matching specific name", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			middlewareCalled := false

			app.Use(middleware.MatchCommandName("/hello"))

			app.Use(func(args bolt.AllMiddlewareArgs) error {
				middlewareCalled = true
				return args.Next()
			})

			// Test matching command (should proceed)
			commandBody := createCommandBodyBuiltin("/hello", "U987654", "C123456")
			event := types.ReceiverEvent{
				Body: commandBody,
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Ack: func(response interface{}) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.True(t, middlewareCalled, "Middleware should be called for matching command")

			// Reset and test with different command (should be ignored)
			middlewareCalled = false
			commandBody = createCommandBodyBuiltin("/goodbye", "U987654", "C123456")
			event.Body = commandBody

			err = app.ProcessEvent(ctx, event)
			require.NoError(t, err)

			assert.False(t, middlewareCalled, "Middleware should NOT be called for non-matching command")
		})
	})
}

// Helper functions for creating test event bodies

func createBlockActionBodyBuiltin(actionID, blockID string) []byte {
	actionBody := map[string]interface{}{
		"type": "block_actions",
		"user": map[string]interface{}{
			"id":   "U123456",
			"name": "testuser",
		},
		"team": map[string]interface{}{
			"id":     "T123456",
			"domain": "test-team",
		},
		"channel": map[string]interface{}{
			"id":   "C123456",
			"name": "general",
		},
		"actions": []interface{}{
			map[string]interface{}{
				"action_id": actionID,
				"block_id":  blockID,
				"type":      "button",
				"value":     "test_value",
			},
		},
		"response_url": "https://hooks.slack.com/actions/T123456/123456789/abcdefg",
		"trigger_id":   "123456789.123456789.abcdefg",
	}

	bodyBytes, _ := json.Marshal(actionBody)
	return bodyBytes
}

func createCommandBody(command, text string) []byte {
	commandBody := map[string]interface{}{
		"token":        "test_token",
		"team_id":      "T123456",
		"team_domain":  "test-team",
		"channel_id":   "C123456",
		"channel_name": "general",
		"user_id":      "U123456",
		"user_name":    "testuser",
		"command":      command,
		"text":         text,
		"response_url": "https://hooks.slack.com/commands/1234/5678",
		"trigger_id":   "13345224609.738474920.8088930838d88f008e0",
	}

	bodyBytes, _ := json.Marshal(commandBody)
	return bodyBytes
}

func createMessageEventBodyBuiltin(userID, channelID, text string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    userID,
			"text":    text,
			"ts":      "1234567890.123456",
			"channel": channelID,
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{userID},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}

func createAppMentionEventBodyBuiltin(userID, channelID, text string) []byte {
	eventBody := map[string]interface{}{
		"token":      "test_token",
		"team_id":    "T123456",
		"api_app_id": "A123456",
		"event": map[string]interface{}{
			"type":    "app_mention",
			"user":    userID,
			"text":    text,
			"ts":      "1234567890.123456",
			"channel": channelID,
		},
		"type":         "event_callback",
		"event_id":     "Ev123456",
		"event_time":   1234567890,
		"authed_users": []string{userID},
	}

	bodyBytes, _ := json.Marshal(eventBody)
	return bodyBytes
}

func createCommandBodyBuiltin(command, userID, channelID string) []byte {
	values := url.Values{}
	values.Set("token", "test_token")
	values.Set("team_id", "T123456")
	values.Set("team_domain", "test-team")
	values.Set("channel_id", channelID)
	values.Set("channel_name", "general")
	values.Set("user_id", userID)
	values.Set("user_name", "testuser")
	values.Set("command", command)
	values.Set("text", "test parameters")
	values.Set("response_url", "https://hooks.slack.com/commands/1234/5678")
	values.Set("trigger_id", "123456789.123456789.abcdefg")

	return []byte(values.Encode())
}
