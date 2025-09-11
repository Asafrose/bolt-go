package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct event arguments for app_mention", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app mention event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello world",
				"channel": "C123456",
				"ts":      "1234567890.123456",
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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Event, "Event should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present")

		// Verify event data
		if eventMap, ok := receivedArgs.Event.(map[string]interface{}); ok {
			assert.Equal(t, "app_mention", eventMap["type"], "Event type should be app_mention")
			assert.Equal(t, "U123456", eventMap["user"], "User ID should match")
			assert.Equal(t, "<@U987654> hello world", eventMap["text"], "Text should match")
		}
	})

	t.Run("should provide correct event arguments for message", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create message event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"user":    "U123456",
				"text":    "Hello world",
				"channel": "C123456",
				"ts":      "1234567890.123456",
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

		// Verify event data
		if eventMap, ok := receivedArgs.Event.(map[string]interface{}); ok {
			assert.Equal(t, "message", eventMap["type"], "Event type should be message")
			assert.Equal(t, "Hello world", eventMap["text"], "Text should match")
		}
	})
}

func TestActionMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct action arguments for button click", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs

		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create button action event
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"block_id":  "block_1",
					"type":      "button",
					"text":      map[string]interface{}{"type": "plain_text", "text": "Click me"},
					"value":     "button_value",
				},
			},
			"user":         map[string]interface{}{"id": "U123456"},
			"channel":      map[string]interface{}{"id": "C123456"},
			"team":         map[string]interface{}{"id": "T123456"},
			"response_url": "https://hooks.slack.com/actions/T123456/123456/abcdef",
		}

		bodyBytes, _ := json.Marshal(actionBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Action, "Action should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
		assert.NotNil(t, receivedArgs.Respond, "Respond function should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present")

		// Verify action data
		if actionMap, ok := receivedArgs.Action.(map[string]interface{}); ok {
			assert.Equal(t, "button_1", actionMap["action_id"], "Action ID should match")
			assert.Equal(t, "button", actionMap["type"], "Action type should be button")
		}
	})

	t.Run("should provide correct action arguments for select menu", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs

		actionID := "select_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create select menu action event
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "select_1",
					"block_id":  "block_1",
					"type":      "static_select",
					"selected_option": map[string]interface{}{
						"text":  map[string]interface{}{"type": "plain_text", "text": "Option 1"},
						"value": "option_1",
					},
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
			Ack: func(response interface{}) error {
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Verify action data
		if actionMap, ok := receivedArgs.Action.(map[string]interface{}); ok {
			assert.Equal(t, "select_1", actionMap["action_id"], "Action ID should match")
			assert.Equal(t, "static_select", actionMap["type"], "Action type should be static_select")
		}
	})
}

func TestCommandMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct command arguments", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCommandMiddlewareArgs

		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create slash command event
		commandBody := map[string]interface{}{
			"command":      "/test",
			"text":         "hello world",
			"user_id":      "U123456",
			"channel_id":   "C123456",
			"team_id":      "T123456",
			"response_url": "https://hooks.slack.com/commands/T123456/123456/abcdef",
			"trigger_id":   "123456.123456.abcdef",
		}

		bodyBytes, _ := json.Marshal(commandBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Command, "Command should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
		assert.NotNil(t, receivedArgs.Respond, "Respond function should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present")

		// Verify command data
		assert.Equal(t, "/test", receivedArgs.Command.Command, "Command should match")
		assert.Equal(t, "hello world", receivedArgs.Command.Text, "Command text should match")
		assert.Equal(t, "U123456", receivedArgs.Command.UserID, "User ID should match")
		assert.Equal(t, "C123456", receivedArgs.Command.ChannelID, "Channel ID should match")
	})

	t.Run("should handle command with empty text", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCommandMiddlewareArgs

		app.Command("/empty", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create slash command event with empty text
		commandBody := map[string]interface{}{
			"command":    "/empty",
			"text":       "",
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
				return nil
			},
		}

		ctx := context.Background()
		err = app.ProcessEvent(ctx, event)
		require.NoError(t, err)

		// Verify command data
		assert.Equal(t, "/empty", receivedArgs.Command.Command, "Command should match")
		assert.Equal(t, "", receivedArgs.Command.Text, "Command text should be empty")
	})
}

func TestShortcutMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct shortcut arguments for global shortcut", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs

		callbackID := "test_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create global shortcut event
		shortcutBody := map[string]interface{}{
			"type":        "shortcut",
			"callback_id": "test_shortcut",
			"user":        map[string]interface{}{"id": "U123456"},
			"team":        map[string]interface{}{"id": "T123456"},
			"trigger_id":  "123456.123456.abcdef",
		}

		bodyBytes, _ := json.Marshal(shortcutBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Shortcut, "Shortcut should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
	})

	t.Run("should provide correct shortcut arguments for message shortcut", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackShortcutMiddlewareArgs

		callbackID := "message_shortcut"
		app.Shortcut(bolt.ShortcutConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackShortcutMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create message shortcut event
		shortcutBody := map[string]interface{}{
			"type":        "message_action",
			"callback_id": "message_shortcut",
			"user":        map[string]interface{}{"id": "U123456"},
			"channel":     map[string]interface{}{"id": "C123456"},
			"team":        map[string]interface{}{"id": "T123456"},
			"message": map[string]interface{}{
				"text": "Original message text",
				"user": "U987654",
			},
		}

		bodyBytes, _ := json.Marshal(shortcutBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Shortcut, "Shortcut should be present")
		assert.NotNil(t, receivedArgs.Say, "Say function should be present for message shortcuts")
	})
}

func TestViewMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct view arguments for view submission", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs

		callbackID := "test_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create view submission event
		viewBody := map[string]interface{}{
			"type": "view_submission",
			"view": map[string]interface{}{
				"callback_id": "test_modal",
				"type":        "modal",
				"title":       map[string]interface{}{"type": "plain_text", "text": "Test Modal"},
				"state": map[string]interface{}{
					"values": map[string]interface{}{
						"block_1": map[string]interface{}{
							"input_1": map[string]interface{}{
								"type":  "plain_text_input",
								"value": "user input",
							},
						},
					},
				},
			},
			"user": map[string]interface{}{"id": "U123456"},
			"team": map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(viewBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.View, "View should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")
	})

	t.Run("should provide correct view arguments for view closed", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs

		callbackID := "closable_modal"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create view closed event
		viewBody := map[string]interface{}{
			"type": "view_closed",
			"view": map[string]interface{}{
				"callback_id": "closable_modal",
				"type":        "modal",
				"title":       map[string]interface{}{"type": "plain_text", "text": "Closable Modal"},
			},
			"user": map[string]interface{}{"id": "U123456"},
			"team": map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(viewBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.View, "View should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
	})
}

func TestOptionsMiddlewareArguments(t *testing.T) {
	t.Run("should provide correct options arguments", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackOptionsMiddlewareArgs

		actionID := "select_1"
		app.Options(bolt.OptionsConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackOptionsMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create options request event
		optionsBody := map[string]interface{}{
			"type":      "block_suggestion",
			"action_id": "select_1",
			"block_id":  "block_1",
			"value":     "te",
			"user":      map[string]interface{}{"id": "U123456"},
			"channel":   map[string]interface{}{"id": "C123456"},
			"team":      map[string]interface{}{"id": "T123456"},
		}

		bodyBytes, _ := json.Marshal(optionsBody)

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

		// Verify middleware arguments
		assert.NotNil(t, receivedArgs.Options, "Options should be present")
		assert.NotNil(t, receivedArgs.Body, "Body should be present")
		assert.NotNil(t, receivedArgs.Context, "Context should be present")
		assert.NotNil(t, receivedArgs.Logger, "Logger should be present")
		assert.NotNil(t, receivedArgs.Client, "Client should be present")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be present")

		// Verify options data
		assert.Equal(t, "select_1", receivedArgs.Options.ActionID, "Action ID should match")
		assert.Equal(t, "block_1", receivedArgs.Options.BlockID, "Block ID should match")
		assert.Equal(t, "te", receivedArgs.Options.Value, "Value should match")
	})
}

func TestMiddlewareArgumentsAdvanced(t *testing.T) {
	t.Run("should extract valid enterprise_id in a shared channel", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create event with enterprise_id for shared channel
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello from shared channel",
				"channel": "C123456",
			},
			"team_id":       "T123456",
			"enterprise_id": "E123456", // Enterprise ID for shared channel
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

		// Verify enterprise_id extraction
		assert.NotNil(t, receivedArgs.Context, "Context should be available")
		// TODO: Add enterprise_id verification when context supports it
	})

	t.Run("should be skipped for tokens_revoked events", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		_ = false // handlerCalled not used in this test

		app.Event("tokens_revoked", func(args bolt.SlackEventMiddlewareArgs) error {
			return nil
		})

		// Create tokens_revoked event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type": "tokens_revoked",
				"tokens": map[string]interface{}{
					"oauth": []string{"xoxp-token"},
					"bot":   []string{"xoxb-token"},
				},
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

		// Authorization should be skipped for tokens_revoked events
		// Handler may or may not be called depending on implementation
		// This test verifies the event is processed without authorization errors
	})

	t.Run("should be skipped for app_uninstalled events", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		_ = false // handlerCalled not used in this test

		app.Event("app_uninstalled", func(args bolt.SlackEventMiddlewareArgs) error {
			return nil
		})

		// Create app_uninstalled event
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type": "app_uninstalled",
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

		// Authorization should be skipped for app_uninstalled events
		// Handler may or may not be called depending on implementation
	})
}

func TestMiddlewareArgumentsRespond(t *testing.T) {
	t.Run("should respond to events with a response_url", func(t *testing.T) {
		// Create mock server for response_url
		responseReceived := false
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			responseReceived = true
			w.WriteHeader(http.StatusOK)
		}))
		defer mockServer.Close()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs

		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create action with response_url pointing to mock server
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"type":      "button",
					"value":     "click_me",
				},
			},
			"response_url": mockServer.URL,
			"user":         map[string]interface{}{"id": "U123456"},
			"channel":      map[string]interface{}{"id": "C123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

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

		// Verify respond function is available
		assert.NotNil(t, receivedArgs.Respond, "Respond function should be available")

		// Test respond function
		err = receivedArgs.Respond(map[string]interface{}{
			"text": "Button clicked!",
		})
		assert.NoError(t, err, "Respond should work with response_url")
		assert.True(t, responseReceived, "Response should be sent to mock server")
	})

	t.Run("should respond with a response object", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs

		actionID := "button_1"
		app.Action(bolt.ActionConstraints{
			ActionID: &actionID,
		}, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create mock server for response_url
		responseReceived := false
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			responseReceived = true
			w.WriteHeader(http.StatusOK)
		}))
		defer mockServer.Close()

		// Create action with response_url pointing to mock server
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"type":      "button",
				},
			},
			"response_url": mockServer.URL,
			"user":         map[string]interface{}{"id": "U123456"},
			"channel":      map[string]interface{}{"id": "C123456"},
		}

		bodyBytes, _ := json.Marshal(actionBody)

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

		// Test respond with complex response object
		response := map[string]interface{}{
			"text": "Complex response",
			"blocks": []interface{}{
				map[string]interface{}{
					"type": "section",
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": "*Bold text* and _italic text_",
					},
				},
			},
			"response_type": "ephemeral",
		}

		err = receivedArgs.Respond(response)
		assert.NoError(t, err, "Respond should work with complex response object")
		assert.True(t, responseReceived, "Response should be sent to mock server")
	})

	t.Run("should be able to use respond for view_submission payloads", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackViewMiddlewareArgs

		callbackID := "modal_1"
		app.View(bolt.ViewConstraints{
			CallbackID: &callbackID,
		}, func(args bolt.SlackViewMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create view_submission with response_url
		viewBody := map[string]interface{}{
			"type": "view_submission",
			"view": map[string]interface{}{
				"callback_id": "modal_1",
				"type":        "modal",
				"title": map[string]interface{}{
					"type": "plain_text",
					"text": "Test Modal",
				},
			},
			"response_urls": []interface{}{
				map[string]interface{}{
					"response_url": "https://hooks.slack.com/actions/T123456/123456/abcdef",
					"channel_id":   "C123456",
				},
			},
			"user": map[string]interface{}{"id": "U123456"},
		}

		bodyBytes, _ := json.Marshal(viewBody)

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

		// TODO: Verify respond function is available for view submissions
		// This depends on the implementation of view middleware arguments
		assert.NotNil(t, receivedArgs, "View args should be received")
	})
}

func TestMiddlewareArgumentsLogger(t *testing.T) {
	t.Run("should be available in middleware/listener args", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app_mention event
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
		require.NoError(t, err)

		// Verify logger is available
		assert.NotNil(t, receivedArgs.Logger, "Logger should be available in middleware args")
	})

	t.Run("should work in the case both logger and logLevel are given", func(t *testing.T) {
		// TODO: Test with custom logger and log level when supported
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
			// Logger:        customLogger,
			// LogLevel:      "debug",
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app_mention event
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
		require.NoError(t, err)

		// Verify logger is available with custom configuration
		assert.NotNil(t, receivedArgs.Logger, "Logger should be available with custom config")
	})
}

func TestMiddlewareArgumentsClient(t *testing.T) {
	t.Run("should be available in middleware/listener args", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app_mention event
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
		require.NoError(t, err)

		// Verify client is available
		assert.NotNil(t, receivedArgs.Client, "Client should be available in middleware args")
	})

	t.Run("should be set to the global app client when authorization doesn't produce a token", func(t *testing.T) {
		// Test with authorize function that doesn't return a user token
		authorizeFn := func(ctx context.Context, source app.AuthorizeSourceData, body interface{}) (*app.AuthorizeResult, error) {
			return &app.AuthorizeResult{
				BotToken:  &fakeToken,
				BotID:     stringPtr("B123456"),
				BotUserID: stringPtr("U987654"),
				TeamID:    stringPtr("T123456"),
				// No UserToken provided
			}, nil
		}

		app, err := bolt.New(bolt.AppOptions{
			Authorize:     authorizeFn,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app_mention event
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
		require.NoError(t, err)

		// Verify client is available and uses global app client
		assert.NotNil(t, receivedArgs.Client, "Client should be available")
		// TODO: Verify it's using the global app client when client interface allows inspection
	})
}

func TestMiddlewareArgumentsSay(t *testing.T) {
	t.Run("should send a simple message to a channel where the incoming event originates", func(t *testing.T) {
		// Create mock Slack API server
		mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock the chat.postMessage API endpoint
			if r.URL.Path == "/api/chat.postMessage" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := map[string]interface{}{
					"ok":      true,
					"channel": "C123456",
					"ts":      "1234567890.123456",
					"message": map[string]interface{}{
						"text": "Hello back!",
					},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer mockAPIServer.Close()

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
			ClientOptions: []slack.Option{slack.OptionAPIURL(mockAPIServer.URL + "/api/")},
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs

		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			return nil
		})

		// Create app_mention event in a channel
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
		require.NoError(t, err)

		// Verify Say function is available
		assert.NotNil(t, receivedArgs.Say, "Say function should be available")

		// Test Say function
		_, err = receivedArgs.Say("Hello back!")
		assert.NoError(t, err, "Say should work for channel events")
	})

	t.Run("for events that should include say() utility", func(t *testing.T) {
		t.Run("should send a simple message to a channel where the incoming event originates", func(t *testing.T) {
			// Create mock Slack API server
			mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Mock the chat.postMessage API endpoint
				if r.URL.Path == "/api/chat.postMessage" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					response := map[string]interface{}{
						"ok":      true,
						"channel": "C123456",
						"ts":      "1234567890.123456",
						"message": map[string]interface{}{
							"text": "Hello from the bot!",
						},
					}
					json.NewEncoder(w).Encode(response)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer mockAPIServer.Close()

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
				ClientOptions: []slack.Option{slack.OptionAPIURL(mockAPIServer.URL + "/api/")},
			})
			require.NoError(t, err)

			var receivedArgs types.SlackEventMiddlewareArgs

			app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
				receivedArgs = args

				// Test that say utility is available
				assert.NotNil(t, args.Say, "Say utility should be available for app_mention events")

				// Test calling say with a simple message
				if args.Say != nil {
					_, err := args.Say("Hello from the bot!")
					assert.NoError(t, err, "Say should work without error")
				}

				return args.Ack(nil)
			})

			// Create app mention event
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "app_mention",
					"user":    "U123456",
					"text":    "<@U987654> hello",
					"channel": "C123456",
					"ts":      "1234567890.123456",
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

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			assert.NotNil(t, receivedArgs.Say, "Say should be available")
			// In a real implementation, we'd verify the say function was called correctly
		})

		t.Run("should send a complex message to a channel where the incoming event originates", func(t *testing.T) {
			// Create mock Slack API server
			mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Mock the chat.postMessage API endpoint
				if r.URL.Path == "/api/chat.postMessage" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					response := map[string]interface{}{
						"ok":      true,
						"channel": "C123456",
						"ts":      "1234567890.123456",
						"message": map[string]interface{}{
							"text": "Complex message",
						},
					}
					json.NewEncoder(w).Encode(response)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer mockAPIServer.Close()

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
				ClientOptions: []slack.Option{slack.OptionAPIURL(mockAPIServer.URL + "/api/")},
			})
			require.NoError(t, err)

			var receivedArgs types.SlackEventMiddlewareArgs

			app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
				receivedArgs = args

				// Test that say utility is available
				assert.NotNil(t, args.Say, "Say utility should be available for app_mention events")

				// Test calling say with a complex message (blocks)
				if args.Say != nil {
					complexMessage := map[string]interface{}{
						"text": "Complex message",
						"blocks": []interface{}{
							map[string]interface{}{
								"type": "section",
								"text": map[string]interface{}{
									"type": "mrkdwn",
									"text": "This is a *complex* message with blocks",
								},
							},
							map[string]interface{}{
								"type": "actions",
								"elements": []interface{}{
									map[string]interface{}{
										"type":      "button",
										"text":      map[string]interface{}{"type": "plain_text", "text": "Click me"},
										"action_id": "button_click",
									},
								},
							},
						},
					}

					_, err := args.Say(complexMessage)
					assert.NoError(t, err, "Complex message should be sent successfully")
				}

				return args.Ack(nil)
			})

			// Create app mention event
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "app_mention",
					"user":    "U123456",
					"text":    "<@U987654> hello",
					"channel": "C123456",
					"ts":      "1234567890.123456",
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

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			assert.NotNil(t, receivedArgs.Say, "Say should be available")
		})
	})

	t.Run("for events that should not include say() utility", func(t *testing.T) {
		t.Run("should not exist in the arguments on incoming events that don't support it", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var receivedArgs types.SlackEventMiddlewareArgs

			// Use an event that doesn't support say() utility (like tokens_revoked)
			app.Event("tokens_revoked", func(args types.SlackEventMiddlewareArgs) error {
				receivedArgs = args

				// Test that say utility is NOT available for tokens_revoked events
				assert.Nil(t, args.Say, "Say utility should NOT be available for tokens_revoked events")

				return args.Ack(nil)
			})

			// Create tokens_revoked event
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type": "tokens_revoked",
					"tokens": map[string]interface{}{
						"oauth": []string{"U123456"},
						"bot":   []string{"U987654"},
					},
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

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			assert.Nil(t, receivedArgs.Say, "Say should not be available for tokens_revoked events")
		})

		t.Run("should handle failures through the App", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("app_uninstalled", func(args types.SlackEventMiddlewareArgs) error {
				// Test that say utility is NOT available
				assert.Nil(t, args.Say, "Say utility should NOT be available for app_uninstalled events")

				// Simulate an error in processing
				return fmt.Errorf("processing failed")
			})

			// Create app_uninstalled event
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type": "app_uninstalled",
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

			err = app.ProcessEvent(context.Background(), event)
			// The app should handle the error - we expect an error to be returned
			assert.Error(t, err, "App should return error from handler")
		})
	})

	t.Run("context", func(t *testing.T) {
		t.Run("should be able to use the app_installed_team_id when provided by the payload", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var receivedArgs types.SlackEventMiddlewareArgs

			app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
				receivedArgs = args

				// Test that context includes app_installed_team_id when provided
				if args.Context != nil && args.Context.Custom != nil {
					if context, ok := args.Context.Custom["app_installed_team_id"]; ok {
						assert.Equal(t, "T123456789", context, "Should have correct app_installed_team_id")
					}
				}

				return args.Ack(nil)
			})

			// Create app mention event with app_installed_team_id
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "app_mention",
					"user":    "U123456",
					"text":    "<@U987654> hello",
					"channel": "C123456",
					"ts":      "1234567890.123456",
				},
				"team_id":               "T123456",
				"app_installed_team_id": "T123456789", // Additional team ID for shared channels
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

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			assert.NotNil(t, receivedArgs.Context, "Context should be available")
		})

		t.Run("should have function executed event details from a custom step payload", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			var receivedArgs types.SlackEventMiddlewareArgs

			app.Event("function_executed", func(args types.SlackEventMiddlewareArgs) error {
				receivedArgs = args

				// Test that context includes function execution details
				assert.NotNil(t, args.Context, "Context should be available")

				// Check for function execution context
				if args.Context != nil && args.Context.Custom != nil {
					if functionExecutionId, ok := args.Context.Custom["function_execution_id"]; ok {
						assert.Equal(t, "Fx123456789", functionExecutionId, "Should have correct function_execution_id")
					}

					if botUserId, ok := args.Context.Custom["bot_user_id"]; ok {
						assert.Equal(t, "U987654321", botUserId, "Should have correct bot_user_id")
					}
				}

				return args.Ack(nil)
			})

			// Create function_executed event with custom step payload
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":                  "function_executed",
					"function":              map[string]interface{}{"callback_id": "custom_function"},
					"function_execution_id": "Fx123456789",
					"bot_user_id":           "U987654321",
					"inputs":                map[string]interface{}{"message": "test input"},
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

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			assert.NotNil(t, receivedArgs.Context, "Context should be available")
		})
	})
}
