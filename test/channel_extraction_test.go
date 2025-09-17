package test

import (
	"context"
	"encoding/json"
	"testing"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelExtractionForReactionEvents(t *testing.T) {
	t.Parallel()

	t.Run("reaction_added events", func(t *testing.T) {
		t.Run("should extract channel from event.item.channel and Say function should work", func(t *testing.T) {
			var channelContextFound bool
			var sayFunctionCalled bool
			var sayError error
			var capturedChannelID string

			// Create test app
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			// Register reaction_added handler that checks channel context
			app.Event(types.EventTypeReactionAdded, func(args bolt.SlackEventMiddlewareArgs) error {
				// Check if channel context was extracted
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}

				// Test Say function - it should not fail with "no channel context"
				sayFunctionCalled = true
				_, sayError = args.Say(types.SayString("Reaction detected!"))
				return nil
			})

			// Create reaction_added event body matching the user's actual event
			eventBody := map[string]interface{}{
				"token":                 "QguweSODHht3QyipSnKZTK2U",
				"team_id":               "T07PJGF1EHY",
				"context_team_id":       "T07PJGF1EHY",
				"context_enterprise_id": nil,
				"api_app_id":            "A08U1HQHAJW",
				"event": map[string]interface{}{
					"type":     "reaction_added",
					"user":     "U085YCVV93R",
					"reaction": "+1",
					"item": map[string]interface{}{
						"type":    "message",
						"channel": "C09F2AV5M9B",
						"ts":      "1758112169.326629",
					},
					"item_user": "U08UCMLFD5F",
					"event_ts":  "1758115405.000200",
				},
				"type":       "event_callback",
				"event_id":   "Ev09FM90HN93",
				"event_time": 1758115405,
				"authorizations": []map[string]interface{}{
					{
						"enterprise_id":         nil,
						"team_id":               "T07PJGF1EHY",
						"user_id":               "U08UCMLFD5F",
						"is_bot":                true,
						"is_enterprise_install": false,
					},
				},
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			// Process the event
			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			// Verify channel context was extracted correctly
			assert.True(t, channelContextFound, "Channel context should have been extracted from event.item.channel")
			assert.Equal(t, "C09F2AV5M9B", capturedChannelID, "Channel ID should match the one from event.item.channel")

			// Verify Say function was called and didn't fail with "no channel context"
			assert.True(t, sayFunctionCalled, "Say function should have been called")
			if sayError != nil {
				assert.NotContains(t, sayError.Error(), "no channel context",
					"Say function should not fail with 'no channel context' error")
			}
		})

		t.Run("should handle reaction_removed events", func(t *testing.T) {
			var channelContextFound bool
			var capturedChannelID string

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event(types.EventTypeReactionRemoved, func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":     "reaction_removed",
					"user":     "U085YCVV93R",
					"reaction": "thumbsdown",
					"item": map[string]interface{}{
						"type":    "message",
						"channel": "C09F2AV5M9B",
						"ts":      "1758112169.326629",
					},
					"item_user": "U08UCMLFD5F",
					"event_ts":  "1758115405.000200",
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.True(t, channelContextFound, "Channel context should have been extracted")
			assert.Equal(t, "C09F2AV5M9B", capturedChannelID, "Channel ID should match")
		})
	})

	t.Run("direct channel events", func(t *testing.T) {
		t.Run("should extract channel from event.channel", func(t *testing.T) {
			var channelContextFound bool
			var capturedChannelID string

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("message", func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":    "message",
					"channel": "C123456789",
					"user":    "U123456789",
					"text":    "Hello world",
					"ts":      "1234567890.123456",
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.True(t, channelContextFound, "Channel context should have been extracted")
			assert.Equal(t, "C123456789", capturedChannelID, "Channel ID should match")
		})
	})

	t.Run("channel_id events", func(t *testing.T) {
		t.Run("should extract channel from event.channel_id", func(t *testing.T) {
			var channelContextFound bool
			var capturedChannelID string

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":       "app_mention",
					"channel_id": "C555666777",
					"user":       "U123456789",
					"text":       "<@U08UCMLFD5F> hello",
					"ts":         "1234567890.123456",
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.True(t, channelContextFound, "Channel context should have been extracted")
			assert.Equal(t, "C555666777", capturedChannelID, "Channel ID should match")
		})
	})

	t.Run("precedence tests", func(t *testing.T) {
		t.Run("should prefer event.channel over event.item.channel when both exist", func(t *testing.T) {
			var channelContextFound bool
			var capturedChannelID string

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("test_event", func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":    "test_event",
					"channel": "C111111111", // Direct channel should take precedence
					"item": map[string]interface{}{
						"channel": "C222222222", // This should be ignored when direct channel exists
					},
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.True(t, channelContextFound, "Channel context should have been extracted")
			assert.Equal(t, "C111111111", capturedChannelID, "Should prefer direct channel")
		})

		t.Run("should prefer event.channel_id over event.item.channel when both exist", func(t *testing.T) {
			var channelContextFound bool
			var capturedChannelID string

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("test_event", func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if channel, exists := args.Context.Custom["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							channelContextFound = true
							capturedChannelID = channelStr
						}
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":       "test_event",
					"channel_id": "C333333333", // channel_id should take precedence over item.channel
					"item": map[string]interface{}{
						"channel": "C444444444",
					},
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.True(t, channelContextFound, "Channel context should have been extracted")
			assert.Equal(t, "C333333333", capturedChannelID, "Should prefer channel_id")
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should handle missing channel gracefully", func(t *testing.T) {
			var channelContextFound bool

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event("team_join", func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if _, exists := args.Context.Custom["channel"]; exists {
						channelContextFound = true
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type": "team_join",
					"user": map[string]interface{}{
						"id":   "U123456789",
						"name": "newuser",
					},
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.False(t, channelContextFound, "Should not have channel context for team_join events")
		})

		t.Run("should handle non-message item types in reaction events", func(t *testing.T) {
			var channelContextFound bool

			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			app.Event(types.EventTypeReactionAdded, func(args bolt.SlackEventMiddlewareArgs) error {
				if args.Context.Custom != nil {
					if _, exists := args.Context.Custom["channel"]; exists {
						channelContextFound = true
					}
				}
				return nil
			})

			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":     "reaction_added",
					"user":     "U085YCVV93R",
					"reaction": "star",
					"item": map[string]interface{}{
						"type": "file", // File reaction, not message
						"file": "F123456789",
						// No channel field for file reactions
					},
					"item_user": "U08UCMLFD5F",
					"event_ts":  "1758115405.000200",
				},
				"type": "event_callback",
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			receiverEvent := types.ReceiverEvent{
				Body: bodyBytes,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Ack: func(response types.AckResponse) error {
					return nil
				},
			}

			ctx := context.Background()
			err = app.ProcessEvent(ctx, receiverEvent)
			require.NoError(t, err)

			assert.False(t, channelContextFound, "Should not have channel context for file reactions")
		})
	})
}

func TestSayFunctionWithChannelContext(t *testing.T) {
	t.Parallel()

	t.Run("should work with reaction_added events", func(t *testing.T) {
		// Create a test app
		testApp, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		var sayFunctionCalled bool
		var sayError error

		// Set up event handler
		testApp.Event(types.EventTypeReactionAdded, func(args types.SlackEventMiddlewareArgs) error {
			sayFunctionCalled = true
			_, sayError = args.Say(types.SayString("Reaction detected!"))
			return nil
		})

		// Create reaction_added event
		eventBody := map[string]interface{}{
			"event": map[string]interface{}{
				"type":     "reaction_added",
				"user":     "U085YCVV93R",
				"reaction": "+1",
				"item": map[string]interface{}{
					"type":    "message",
					"channel": "C09F2AV5M9B",
					"ts":      "1758112169.326629",
				},
				"item_user": "U08UCMLFD5F",
				"event_ts":  "1758115405.000200",
			},
			"type": "event_callback",
		}

		bodyBytes, err := json.Marshal(eventBody)
		require.NoError(t, err)

		receiverEvent := types.ReceiverEvent{
			Body: bodyBytes,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Ack: func(response types.AckResponse) error {
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		err = testApp.ProcessEvent(ctx, receiverEvent)

		// Should not fail due to missing channel context
		require.NoError(t, err)
		assert.True(t, sayFunctionCalled, "Say function should have been called")

		// Say function should not return "no channel context" error
		if sayError != nil {
			assert.NotContains(t, sayError.Error(), "no channel context",
				"Say function should not fail with 'no channel context' error")
		}
	})
}
