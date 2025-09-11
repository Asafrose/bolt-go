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

func TestRegExpActionRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a block action event to a handler registered with action(RegExp) that matches the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Create RegExp pattern for action IDs starting with "btn_"
		actionPattern := regexp.MustCompile(`^btn_.*`)

		constraints := bolt.ActionConstraints{
			ActionIDPattern: actionPattern,
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create action event with matching action_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "btn_submit",
					"block_id":  "block_1",
					"type":      "button",
					"value":     "submit",
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

		assert.True(t, handlerCalled, "Handler should have been called for matching RegExp")
		if actionMap, ok := ExtractRawActionData(receivedArgs.Action); ok {
			assert.Equal(t, "btn_submit", actionMap["action_id"], "Action ID should match")
		}
	})

	t.Run("should NOT route a block action event to a handler registered with action(RegExp) that does NOT match the action ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Create RegExp pattern for action IDs starting with "btn_"
		actionPattern := regexp.MustCompile(`^btn_.*`)

		constraints := bolt.ActionConstraints{
			ActionIDPattern: actionPattern,
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create action event with NON-matching action_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "select_menu_1", // Does NOT match ^btn_.*
					"block_id":  "block_1",
					"type":      "static_select",
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching RegExp")
	})

	t.Run("should route a block action event to a handler registered with action({block_id: RegExp}) that matches the block ID", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackActionMiddlewareArgs
		handlerCalled := false

		// Create RegExp pattern for block IDs ending with "_section"
		blockPattern := regexp.MustCompile(`.*_section$`)

		constraints := bolt.ActionConstraints{
			BlockIDPattern: blockPattern,
		}

		app.Action(constraints, func(args bolt.SlackActionMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create action event with matching block_id
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"block_id":  "header_section", // Matches .*_section$
					"type":      "button",
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

		assert.True(t, handlerCalled, "Handler should have been called for matching block RegExp")
		if actionMap, ok := ExtractRawActionData(receivedArgs.Action); ok {
			assert.Equal(t, "header_section", actionMap["block_id"], "Block ID should match")
		}
	})
}

func TestRegExpCommandRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a command to a handler registered with command(RegExp) if command name matches", func(t *testing.T) {
		// TODO: Add RegExp support for commands
		// This test is a placeholder for when we implement Command RegExp support
		t.Skip("Command RegExp support not yet implemented")
	})
}

func TestRegExpEventRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a Slack event to a handler registered with event(RegExp)", func(t *testing.T) {
		// TODO: Add RegExp support for events
		// This test is a placeholder for when we implement Event RegExp support
		t.Skip("Event RegExp support not yet implemented")
	})
}

func TestRegExpMessageRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a message event to a handler registered with message(RegExp) if message contents match", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackEventMiddlewareArgs
		handlerCalled := false

		// Create RegExp pattern for messages containing "hello"
		messagePattern := regexp.MustCompile(`(?i)hello`)

		app.Message(messagePattern, func(args bolt.SlackEventMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			// Debug: print message details
			t.Logf("Message field: %+v", args.Message)
			if args.Message != nil {
				t.Logf("Message text: %s", args.Message.Text)
			}
			t.Logf("Event field: %+v", args.Event)
			return nil
		})

		// Create message event with matching text
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"text":    "Hello there!",
				"user":    "U123456",
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

		assert.True(t, handlerCalled, "Handler should have been called for matching message RegExp")
		assert.NotNil(t, receivedArgs.Message, "Message should be available")
	})

	t.Run("should NOT route a message event to a handler registered with message(RegExp) if message contents do NOT match", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Create RegExp pattern for messages containing "hello"
		messagePattern := regexp.MustCompile(`(?i)hello`)

		app.Message(messagePattern, func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create message event with NON-matching text
		eventBody := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "message",
				"text":    "Good morning!", // Does NOT contain "hello"
				"user":    "U123456",
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching message RegExp")
	})
}

func TestRegExpShortcutRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a Slack shortcut event to a handler registered with shortcut(RegExp) that matches the callback ID", func(t *testing.T) {
		// TODO: Add RegExp support for shortcuts
		// This test is a placeholder for when we implement Shortcut RegExp support
		t.Skip("Shortcut RegExp support not yet implemented")
	})
}

func TestRegExpViewRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a view submission event to a handler registered with view(RegExp) that matches the callback ID", func(t *testing.T) {
		// TODO: Add RegExp support for views
		// This test is a placeholder for when we implement View RegExp support
		t.Skip("View RegExp support not yet implemented")
	})
}

func TestRegExpOptionsRouting(t *testing.T) {
	t.Parallel()
	t.Run("should route a block suggestion event to a handler registered with options(RegExp) that matches the action ID", func(t *testing.T) {
		// TODO: Add RegExp support for options
		// This test is a placeholder for when we implement Options RegExp support
		t.Skip("Options RegExp support not yet implemented")
	})
}
