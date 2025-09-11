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

// TestCommandRouting implements the missing tests from routing-command.spec.ts
func TestCommandRouting(t *testing.T) {
	t.Run("should route a command to a handler registered with command(string) if command name matches", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCommandMiddlewareArgs
		handlerCalled := false

		// Register handler with string command
		app.Command("/test-command", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create command event with matching command
		commandBody := map[string]interface{}{
			"token":        "test_token",
			"team_id":      "T123456",
			"team_domain":  "test-team",
			"channel_id":   "C123456",
			"channel_name": "general",
			"user_id":      "U123456",
			"user_name":    "testuser",
			"command":      "/test-command",
			"text":         "hello world",
			"response_url": "https://hooks.slack.com/commands/1234/5678",
			"trigger_id":   "13345224609.738474920.8088930838d88f008e0",
		}

		bodyBytes, _ := json.Marshal(commandBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for matching command")
		assert.Equal(t, "/test-command", receivedArgs.Command.Command, "Command should match")
		assert.Equal(t, "hello world", receivedArgs.Command.Text, "Command text should be available")
	})

	t.Run("should route a command to a handler registered with command(RegExp) if command name matches", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCommandMiddlewareArgs
		handlerCalled := false

		// Register handler with RegExp command pattern
		commandPattern := regexp.MustCompile(`^/admin-.*`)
		app.CommandPattern(commandPattern, func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true
			return nil
		})

		// Create command event with matching command
		commandBody := map[string]interface{}{
			"token":        "test_token",
			"team_id":      "T123456",
			"team_domain":  "test-team",
			"channel_id":   "C123456",
			"channel_name": "general",
			"user_id":      "U123456",
			"user_name":    "testuser",
			"command":      "/admin-deploy", // Should match /admin-.* pattern
			"text":         "production",
			"response_url": "https://hooks.slack.com/commands/1234/5678",
			"trigger_id":   "13345224609.738474920.8088930838d88f008e0",
		}

		bodyBytes, _ := json.Marshal(commandBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called for RegExp matching command")
		assert.Equal(t, "/admin-deploy", receivedArgs.Command.Command, "Command should match")
		assert.Equal(t, "production", receivedArgs.Command.Text, "Command text should be available")
	})

	t.Run("should route a command to the corresponding handler and only acknowledge in the handler", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		var receivedArgs bolt.SlackCommandMiddlewareArgs
		handlerCalled := false
		ackCalled := false

		app.Command("/ack-test", func(args bolt.SlackCommandMiddlewareArgs) error {
			receivedArgs = args
			handlerCalled = true

			// Call ack in the handler
			response := bolt.CommandResponse{
				Text:         "Command acknowledged!",
				ResponseType: "ephemeral",
			}
			err := args.Ack(&response)
			if err == nil {
				ackCalled = true
			}
			return err
		})

		// Create command event
		commandBody := map[string]interface{}{
			"token":        "test_token",
			"team_id":      "T123456",
			"channel_id":   "C123456",
			"user_id":      "U123456",
			"command":      "/ack-test",
			"text":         "",
			"response_url": "https://hooks.slack.com/commands/1234/5678",
			"trigger_id":   "13345224609.738474920.8088930838d88f008e0",
		}

		bodyBytes, _ := json.Marshal(commandBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.True(t, ackCalled, "Ack should have been called in handler")
		assert.NotNil(t, receivedArgs.Ack, "Ack function should be available")
		assert.Equal(t, "/ack-test", receivedArgs.Command.Command, "Command should match")
	})

	t.Run("should not execute handler if no routing found", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		handlerCalled := false

		// Register handler for different command
		app.Command("/other-command", func(args bolt.SlackCommandMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Create command event that doesn't match
		commandBody := map[string]interface{}{
			"token":        "test_token",
			"team_id":      "T123456",
			"channel_id":   "C123456",
			"user_id":      "U123456",
			"command":      "/nonexistent-command", // Different command
			"text":         "",
			"response_url": "https://hooks.slack.com/commands/1234/5678",
			"trigger_id":   "13345224609.738474920.8088930838d88f008e0",
		}

		bodyBytes, _ := json.Marshal(commandBody)

		event := types.ReceiverEvent{
			Body: bodyBytes,
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

		assert.False(t, handlerCalled, "Handler should NOT have been called for non-matching command")
	})
}
