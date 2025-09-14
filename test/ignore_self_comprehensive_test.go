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

// TestIgnoreSelfComprehensive implements the missing tests from ignore-self.spec.ts
func TestIgnoreSelfComprehensive(t *testing.T) {
	t.Parallel()
	t.Run("with ignoreSelf true (default)", func(t *testing.T) {
		t.Run("should ack & ignore message events identified as a bot message from the same bot ID as this app", func(t *testing.T) {
			// Test with default ignoreSelf=true behavior
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
				BotID:         &fakeBotID,
				BotUserID:     &fakeBotUserID,
				// ignoreSelf defaults to true
			})
			require.NoError(t, err)

			handlerCalled := false
			app.Message("hello", func(args types.SlackEventMiddlewareArgs) error {
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create a bot message from the same bot ID as this app
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "message",
					"subtype": "bot_message",
					"text":    "hello from bot",
					"channel": "C123456",
					"ts":      "1234567890.123456",
					"bot_id":  fakeBotID, // Same bot ID as this app
					"user":    fakeBotUserID,
				},
				"team_id": "T123456",
			}

			bodyBytes, _ := json.Marshal(eventBody)

			event := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler should NOT be called because the message is from the same bot
			assert.False(t, handlerCalled, "Handler should not be called for messages from same bot ID")
		})

		t.Run("should ack & ignore events that match own app", func(t *testing.T) {
			// Test with default ignoreSelf=true behavior
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
				BotID:         &fakeBotID,
				BotUserID:     &fakeBotUserID,
				// ignoreSelf defaults to true
			})
			require.NoError(t, err)

			handlerCalled := false
			app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create an app mention from the same app
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "app_mention",
					"text":    "hello from app",
					"channel": "C123456",
					"ts":      "1234567890.123456",
					"user":    fakeBotUserID, // Same user as this app's bot
				},
				"team_id": "T123456",
			}

			bodyBytes, _ := json.Marshal(eventBody)

			event := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler should NOT be called because the event is from the same app
			assert.False(t, handlerCalled, "Handler should not be called for events from same app")
		})

		t.Run("should not filter `member_joined_channel` and `member_left_channel` events originating from own app", func(t *testing.T) {
			// Test with default ignoreSelf=true behavior
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
				// ignoreSelf defaults to true
			})
			require.NoError(t, err)

			joinHandlerCalled := false
			leaveHandlerCalled := false

			app.Event("member_joined_channel", func(args types.SlackEventMiddlewareArgs) error {
				joinHandlerCalled = true
				return args.Ack(nil)
			})

			app.Event("member_left_channel", func(args types.SlackEventMiddlewareArgs) error {
				leaveHandlerCalled = true
				return args.Ack(nil)
			})

			// Test member_joined_channel event from own app
			joinEventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "member_joined_channel",
					"user":    fakeBotUserID, // Same user as this app's bot
					"channel": "C123456",
					"ts":      "1234567890.123456",
				},
				"team_id": "T123456",
			}

			bodyBytes, _ := json.Marshal(joinEventBody)

			event := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler SHOULD be called for member_joined_channel even from own app
			assert.True(t, joinHandlerCalled, "Handler should be called for member_joined_channel from own app")

			// Test member_left_channel event from own app
			leaveEventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "member_left_channel",
					"user":    fakeBotUserID, // Same user as this app's bot
					"channel": "C123456",
					"ts":      "1234567890.123456",
				},
				"team_id": "T123456",
			}

			bodyBytes, _ = json.Marshal(leaveEventBody)

			event = types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler SHOULD be called for member_left_channel even from own app
			assert.True(t, leaveHandlerCalled, "Handler should be called for member_left_channel from own app")
		})
	})

	t.Run("with ignoreSelf false", func(t *testing.T) {
		t.Run("should ack & route message events identified as a bot message from the same bot ID as this app to the handler", func(t *testing.T) {
			// Test with ignoreSelf=false
			ignoreSelf := false
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
				BotID:         &fakeBotID,
				BotUserID:     &fakeBotUserID,
				IgnoreSelf:    &ignoreSelf, // Explicitly set to false
			})
			require.NoError(t, err)

			handlerCalled := false
			app.Message("hello", func(args types.SlackEventMiddlewareArgs) error {
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create a bot message from the same bot ID as this app
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "message",
					"subtype": "bot_message",
					"text":    "hello from bot",
					"channel": "C123456",
					"ts":      "1234567890.123456",
					"bot_id":  fakeBotID, // Same bot ID as this app
					"user":    fakeBotUserID,
				},
				"team_id": "T123456",
			}

			bodyBytes, _ := json.Marshal(eventBody)

			event := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler SHOULD be called because ignoreSelf is false
			assert.True(t, handlerCalled, "Handler should be called for messages from same bot ID when ignoreSelf=false")
		})

		t.Run("should ack & route events that match own app", func(t *testing.T) {
			// Test with ignoreSelf=false
			ignoreSelf := false
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
				BotID:         &fakeBotID,
				BotUserID:     &fakeBotUserID,
				IgnoreSelf:    &ignoreSelf, // Explicitly set to false
			})
			require.NoError(t, err)

			handlerCalled := false
			app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
				handlerCalled = true
				return args.Ack(nil)
			})

			// Create an app mention from the same app
			eventBody := map[string]interface{}{
				"type": "event_callback",
				"event": map[string]interface{}{
					"type":    "app_mention",
					"text":    "hello from app",
					"channel": "C123456",
					"ts":      "1234567890.123456",
					"user":    fakeBotUserID, // Same user as this app's bot
				},
				"team_id": "T123456",
			}

			bodyBytes, _ := json.Marshal(eventBody)

			event := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			err = app.ProcessEvent(context.Background(), event)
			require.NoError(t, err)

			// Handler SHOULD be called because ignoreSelf is false
			assert.True(t, handlerCalled, "Handler should be called for events from same app when ignoreSelf=false")
		})
	})
}
