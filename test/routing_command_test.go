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

// Test helpers for command routing
func createSlashCommandBody(command, text string) []byte {
	cmd := map[string]interface{}{
		"token":        "verification-token",
		"team_id":      "T123456",
		"team_domain":  "testteam",
		"channel_id":   "C123456",
		"channel_name": "general",
		"user_id":      "U123456",
		"user_name":    "testuser",
		"command":      command,
		"text":         text,
		"response_url": "https://hooks.slack.com/commands/T123456/123456/abcdef",
		"trigger_id":   "123456.123456.abcdef",
		"api_app_id":   "A123456",
	}

	body, _ := json.Marshal(cmd)
	return body
}

func TestAppCommandRouting(t *testing.T) {
	t.Parallel()
	t.Run("should have channel context for say function", func(t *testing.T) {
		var capturedChannelID string
		var sayError error
		var sayFunctionCalled bool

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler that checks Say function context
		app.Command("/asaf-test", func(args bolt.SlackCommandMiddlewareArgs) error {
			// Capture the channel ID from the command
			capturedChannelID = args.Command.ChannelID

			// Check if the Say function has channel context by inspecting the context
			if args.Context.Custom != nil {
				if channel, exists := args.Context.Custom["channel"]; exists {
					if _, ok := channel.(string); ok {
						sayFunctionCalled = true
						// Don't actually call Say to avoid auth errors
						sayError = nil
						return nil
					}
				}
			}

			sayError = errors.New("no channel context for say function")
			return nil
		})

		// Create receiver event with the exact payload from the user's example
		cmdBody := map[string]interface{}{
			"token":                 "QguweSODHht3QyipSnKZTK2U",
			"team_id":               "T07PJGF1EHY",
			"team_domain":           "daylightsec",
			"channel_id":            "C09F2AV5M9B",
			"channel_name":          "asaf-test-channel-2",
			"user_id":               "U085YCVV93R",
			"user_name":             "asaf",
			"command":               "/asaf-test",
			"text":                  "",
			"api_app_id":            "A08U1HQHAJW",
			"is_enterprise_install": "false",
			"response_url":          "https://hooks.slack.com/commands/T07PJGF1EHY/9547437712641/xkzAB8pePel7ds0dPVPcKzAw",
			"trigger_id":            "9547437736881.7800559048610.714634ca4c861162e2fc7e2abe2eb3ad",
		}

		body, _ := json.Marshal(cmdBody)
		event := types.ReceiverEvent{
			Body: body,
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

		// Verify channel ID was captured correctly
		assert.Equal(t, "C09F2AV5M9B", capturedChannelID)

		// Verify channel context was available for Say function
		assert.True(t, sayFunctionCalled, "Say function should have channel context available")
		assert.NoError(t, sayError, "Say function should not return 'no channel context' error")
	})

	t.Run("should work with actual say function call using SayString", func(t *testing.T) {
		var capturedChannelID string
		var sayError error

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler that actually calls Say with SayString (the original failing case)
		app.Command("/asaf-test", func(args bolt.SlackCommandMiddlewareArgs) error {
			capturedChannelID = args.Command.ChannelID

			// This is the exact same call from the user's original code that was failing
			_, err := args.Say(types.SayString("You have triggered the command"))
			sayError = err
			return nil
		})

		// Create receiver event with the exact payload from the user's example
		cmdBody := map[string]interface{}{
			"token":                 "QguweSODHht3QyipSnKZTK2U",
			"team_id":               "T07PJGF1EHY",
			"team_domain":           "daylightsec",
			"channel_id":            "C09F2AV5M9B",
			"channel_name":          "asaf-test-channel-2",
			"user_id":               "U085YCVV93R",
			"user_name":             "asaf",
			"command":               "/asaf-test",
			"text":                  "",
			"api_app_id":            "A08U1HQHAJW",
			"is_enterprise_install": "false",
			"response_url":          "https://hooks.slack.com/commands/T07PJGF1EHY/9547437712641/xkzAB8pePel7ds0dPVPcKzAw",
			"trigger_id":            "9547437736881.7800559048610.714634ca4c861162e2fc7e2abe2eb3ad",
		}

		body, _ := json.Marshal(cmdBody)
		event := types.ReceiverEvent{
			Body: body,
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

		// Verify channel ID was captured correctly
		assert.Equal(t, "C09F2AV5M9B", capturedChannelID)

		// The Say function should fail with invalid_auth, not "no channel context"
		// This proves that the channel context is working and it's trying to make the API call
		if sayError != nil {
			assert.Contains(t, sayError.Error(), "invalid_auth", "Should fail with auth error, not channel context error")
		}
	})

	t.Run("should route slash command to handler", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/test", "hello world"),
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

		assert.True(t, handlerCalled, "Command handler should have been called")
	})

	t.Run("should not route command if name doesn't match", func(t *testing.T) {
		handlerCalled := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler for different command
		app.Command("/different", func(args bolt.SlackCommandMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/test", "hello world"),
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

		assert.False(t, handlerCalled, "Command handler should not have been called")
	})

	t.Run("should pass correct command data to handler", func(t *testing.T) {
		var receivedCommand bolt.SlashCommand
		var receivedBody interface{}

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler that captures command data
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedCommand = args.Command
			receivedBody = args.Body
			assert.NotNil(t, args.Context, "Context should be present")
			assert.NotNil(t, args.Logger, "Logger should be present")
			assert.NotNil(t, args.Client, "Client should be present")
			assert.NotNil(t, args.Ack, "Ack function should be present")
			assert.NotNil(t, args.Say, "Say function should be present")
			assert.NotNil(t, args.Respond, "Respond function should be present")
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/test", "hello world"),
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

		assert.NotNil(t, receivedBody, "Body data should have been passed to handler")

		// Verify command structure
		assert.Equal(t, "/test", receivedCommand.Command, "Command should be correct")
		assert.Equal(t, "hello world", receivedCommand.Text, "Command text should be correct")
		assert.Equal(t, "U123456", receivedCommand.UserID, "User ID should be correct")
		assert.Equal(t, "C123456", receivedCommand.ChannelID, "Channel ID should be correct")
		assert.Equal(t, "T123456", receivedCommand.TeamID, "Team ID should be correct")
	})

	t.Run("should handle multiple handlers for same command", func(t *testing.T) {
		handler1Called := false
		handler2Called := false

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register multiple handlers for same command
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			handler1Called = true
			return nil
		})

		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			handler2Called = true
			return nil
		})

		// Create receiver event
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/test", "hello world"),
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

		assert.True(t, handler1Called, "First command handler should have been called")
		assert.True(t, handler2Called, "Second command handler should have been called")
	})

	t.Run("should handle commands with different text", func(t *testing.T) {
		var receivedText string

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler
		app.Command("/echo", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedText = args.Command.Text
			return nil
		})

		// Create receiver event with specific text
		testText := "this is a test message"
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/echo", testText),
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

		assert.Equal(t, testText, receivedText, "Command text should be passed correctly")
	})

	t.Run("should handle empty command text", func(t *testing.T) {
		handlerCalled := false
		var receivedText string

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register command handler
		app.Command("/test", func(args bolt.SlackCommandMiddlewareArgs) error {
			handlerCalled = true
			receivedText = args.Command.Text
			return nil
		})

		// Create receiver event with empty text
		event := types.ReceiverEvent{
			Body: createSlashCommandBody("/test", ""),
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

		assert.True(t, handlerCalled, "Command handler should have been called")
		assert.Equal(t, "", receivedText, "Empty command text should be handled correctly")
	})
}
