package test

import (
	"log/slog"
	"regexp"
	"testing"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/middleware"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuiltinComprehensive implements all missing tests from builtin.spec.ts
func TestBuiltinComprehensive(t *testing.T) {
	t.Parallel()

	// Test data
	fakeBotUserId := "B123456"

	t.Run("matchMessage", func(t *testing.T) {
		t.Run("using a string pattern", func(t *testing.T) {
			pattern := "foo"
			matchingText := "foobar"
			nonMatchingText := "bar"

			t.Run("should match message events with a pattern that matches", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgs(matchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was called
				assert.True(t, args.Context.Custom["nextCalled"].(bool))
			})

			t.Run("should match app_mention events with a pattern that matches", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyAppMentionArgs(matchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was called
				assert.True(t, args.Context.Custom["nextCalled"].(bool))
			})

			t.Run("should filter out message events with a pattern that does not match", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgs(nonMatchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})

			t.Run("should filter out app_mention events with a pattern that does not match", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyAppMentionArgs(nonMatchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})

			t.Run("should filter out message events which do not have text (block kit)", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgsWithBlocks(ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})
		})

		t.Run("using a RegExp pattern", func(t *testing.T) {
			pattern := regexp.MustCompile("foo")
			matchingText := "foobar"
			nonMatchingText := "bar"

			t.Run("should match message events with a pattern that matches", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgs(matchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was called and matches were set
				assert.True(t, args.Context.Custom["nextCalled"].(bool))
				assert.NotNil(t, ctx.Custom["matches"])
			})

			t.Run("should match app_mention events with a pattern that matches", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyAppMentionArgs(matchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was called and matches were set
				assert.True(t, args.Context.Custom["nextCalled"].(bool))
				assert.NotNil(t, ctx.Custom["matches"])
			})

			t.Run("should filter out message events with a pattern that does not match", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgs(nonMatchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})

			t.Run("should filter out app_mention events with a pattern that does not match", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyAppMentionArgs(nonMatchingText, ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})

			t.Run("should filter out message events which do not have text (block kit)", func(t *testing.T) {
				middleware := middleware.MatchMessage(pattern)
				ctx := &types.Context{IsEnterpriseInstall: false}
				args := createDummyMessageArgsWithBlocks(ctx)

				err := middleware(args)
				require.NoError(t, err)

				// Verify Next was NOT called
				nextCalled, exists := args.Context.Custom["nextCalled"]
				if exists {
					assert.False(t, nextCalled.(bool))
				}
			})
		})
	})

	t.Run("directMention", func(t *testing.T) {
		t.Run("should bail when the context does not provide a bot user ID", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false} // No BotUserID
			args := createDummyMessageArgs("hello", ctx)

			err := middleware.DirectMention()(args)

			// Should return an error about missing bot user ID
			require.Error(t, err)
			var contextErr *errors.ContextMissingPropertyError
			require.ErrorAs(t, err, &contextErr)
			assert.Equal(t, "botUserId", contextErr.MissingProperty)
		})

		t.Run("should match message events that mention the bot user ID at the beginning of message text", func(t *testing.T) {
			messageText := "<@" + fakeBotUserId + "> hi"
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyMessageArgs(messageText, ctx)

			err := middleware.DirectMention()(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should not match message events that do not mention the bot user ID", func(t *testing.T) {
			messageText := "hi"
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyMessageArgs(messageText, ctx)

			err := middleware.DirectMention()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should not match message events that mention the bot user ID NOT at the beginning of message text", func(t *testing.T) {
			messageText := "hi <@" + fakeBotUserId + "> "
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyMessageArgs(messageText, ctx)

			err := middleware.DirectMention()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should not match message events which do not have text (block kit)", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyMessageArgsWithBlocks(ctx)

			err := middleware.DirectMention()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should not match message events that contain a link to a conversation at the beginning", func(t *testing.T) {
			messageText := "<#C1234> hi"
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyMessageArgs(messageText, ctx)

			err := middleware.DirectMention()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("ignoreSelf", func(t *testing.T) {
		t.Run("should continue middleware processing for non-event payloads", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
				BotID:               &fakeBotUserId,
			}
			args := createDummyCommandArgs(ctx)

			err := middleware.IgnoreSelf()(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should ignore message events identified as a bot message from the same bot ID as this app", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
				BotID:               &fakeBotUserId,
			}
			args := createDummyBotMessageArgs(fakeBotUserId, ctx)

			err := middleware.IgnoreSelf()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should ignore events with only a botUserId", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
			}
			args := createDummyReactionAddedArgs(fakeBotUserId, ctx)

			err := middleware.IgnoreSelf()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should ignore events that match own app", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
				BotID:               &fakeBotUserId,
			}
			args := createDummyReactionAddedArgs(fakeBotUserId, ctx)

			err := middleware.IgnoreSelf()(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should not filter member_joined_channel and member_left_channel events originating from own app", func(t *testing.T) {
			ctx := &types.Context{
				IsEnterpriseInstall: false,
				BotUserID:           &fakeBotUserId,
				BotID:               &fakeBotUserId,
			}

			// Test member_joined_channel
			args1 := createDummyMemberChannelArgs("member_joined_channel", fakeBotUserId, ctx)
			err := middleware.IgnoreSelf()(args1)
			require.NoError(t, err)
			nextCalled1, exists1 := args1.Context.Custom["nextCalled"]
			assert.True(t, exists1 && nextCalled1.(bool))

			// Test member_left_channel
			args2 := createDummyMemberChannelArgs("member_left_channel", fakeBotUserId, ctx)
			err = middleware.IgnoreSelf()(args2)
			require.NoError(t, err)
			nextCalled2, exists2 := args2.Context.Custom["nextCalled"]
			assert.True(t, exists2 && nextCalled2.(bool))
		})
	})

	t.Run("onlyCommands", func(t *testing.T) {
		t.Run("should continue middleware processing for a command payload", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyCommandArgs(ctx)

			err := middleware.OnlyCommands(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should ignore non-command payloads", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyReactionAddedArgs("U123456", ctx)

			err := middleware.OnlyCommands(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("matchCommandName", func(t *testing.T) {
		t.Run("should continue middleware processing for requests that match exactly", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyCommandArgsWithName("/hi", ctx)

			middleware := middleware.MatchCommandName("/hi")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should continue middleware processing for requests that match a pattern", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyCommandArgsWithName("/hi", ctx)

			middleware := middleware.MatchCommandName(regexp.MustCompile("h"))
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should skip other requests", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyCommandArgsWithName("/hi", ctx)

			middleware := middleware.MatchCommandName("/will-not-match")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("onlyEvents", func(t *testing.T) {
		t.Run("should continue middleware processing for valid requests", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyAppMentionArgs("hello", ctx)

			err := middleware.OnlyEvents(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should skip other requests", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyCommandArgsWithName("/hi", ctx)

			err := middleware.OnlyEvents(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("matchEventType", func(t *testing.T) {
		t.Run("should continue middleware processing for when event type matches", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyAppMentionArgs("hello", ctx)

			middleware := middleware.MatchEventType("app_mention")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should continue middleware processing for if RegExp match occurs on event type", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}

			// Test app_mention
			args1 := createDummyAppMentionArgs("hello", ctx)
			middleware := middleware.MatchEventType(regexp.MustCompile("app_mention|app_home_opened"))
			err := middleware(args1)
			require.NoError(t, err)
			nextCalled1, exists1 := args1.Context.Custom["nextCalled"]
			assert.True(t, exists1 && nextCalled1.(bool))

			// Test app_home_opened
			args2 := createDummyAppHomeOpenedArgs(ctx)
			err = middleware(args2)
			require.NoError(t, err)
			nextCalled2, exists2 := args2.Context.Custom["nextCalled"]
			assert.True(t, exists2 && nextCalled2.(bool))
		})

		t.Run("should skip non-matching event types", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyAppMentionArgs("hello", ctx)

			middleware := middleware.MatchEventType("app_home_opened")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})

		t.Run("should skip non-matching event types via RegExp", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyAppMentionArgs("hello", ctx)

			middleware := middleware.MatchEventType(regexp.MustCompile("foo"))
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("subtype", func(t *testing.T) {
		t.Run("should continue middleware processing for match message subtypes", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyBotMessageArgs("B1234", ctx)

			middleware := middleware.Subtype("bot_message")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			assert.True(t, exists && nextCalled.(bool))
		})

		t.Run("should skip non-matching message subtypes", func(t *testing.T) {
			ctx := &types.Context{IsEnterpriseInstall: false}
			args := createDummyBotMessageArgs("B1234", ctx)

			middleware := middleware.Subtype("me_message")
			err := middleware(args)
			require.NoError(t, err)

			// Verify Next was NOT called
			nextCalled, exists := args.Context.Custom["nextCalled"]
			if exists {
				assert.False(t, nextCalled.(bool))
			}
		})
	})

	t.Run("isSlackEventMiddlewareArgsOptions", func(t *testing.T) {
		t.Run("should return true if object is SlackEventMiddlewareArgsOptions", func(t *testing.T) {
			options := middleware.SlackEventMiddlewareArgsOptions{AutoAcknowledge: true}
			result := middleware.IsSlackEventMiddlewareArgsOptions(options)
			assert.True(t, result)
		})

		t.Run("should narrow proper type if object is SlackEventMiddlewareArgsOptions", func(t *testing.T) {
			options := map[string]interface{}{"autoAcknowledge": true}
			if middleware.IsSlackEventMiddlewareArgsOptions(options) {
				// In Go, we can't do compile-time type narrowing like TypeScript,
				// but we can verify the function correctly identifies the type
				// Success - the type was correctly identified
			} else {
				assert.Fail(t, "Should be identified as SlackEventMiddlewareArgsOptions")
			}
		})

		t.Run("should return false if object is Middleware", func(t *testing.T) {
			middlewareFunc := func(args types.AllMiddlewareArgs) error { return nil }
			result := middleware.IsSlackEventMiddlewareArgsOptions(middlewareFunc)
			assert.False(t, result)
		})
	})
}

// Helper functions to create test data

func createDummyMessageArgs(text string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    "message",
		"text":    text,
		"user":    "U123456",
		"channel": "C123456",
		"ts":      "1234567890.123456",
	}

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: func() types.SlackEvent { parsedEvent, _ := helpers.ParseSlackEvent(eventData); return parsedEvent }(),
		Message: &types.MessageEvent{
			MessageEvent: slackevents.MessageEvent{
				Text:      text,
				User:      "U123456",
				Channel:   "C123456",
				TimeStamp: "1234567890.123456",
			},
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyAppMentionArgs(text string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    "app_mention",
		"text":    text,
		"user":    "U123456",
		"channel": "C123456",
		"ts":      "1234567890.123456",
	}

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: func() types.SlackEvent { parsedEvent, _ := helpers.ParseSlackEvent(eventData); return parsedEvent }(),
		Message: &types.MessageEvent{
			MessageEvent: slackevents.MessageEvent{
				Text:      text,
				User:      "U123456",
				Channel:   "C123456",
				TimeStamp: "1234567890.123456",
			},
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyMessageArgsWithBlocks(ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    "message",
		"text":    "", // Empty text
		"user":    "U123456",
		"channel": "C123456",
		"ts":      "1234567890.123456",
		"blocks": []interface{}{
			map[string]interface{}{
				"type": "divider",
			},
		},
	}

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: func() types.SlackEvent { parsedEvent, _ := helpers.ParseSlackEvent(eventData); return parsedEvent }(),
		Message: &types.MessageEvent{
			MessageEvent: slackevents.MessageEvent{
				Text:      "", // Empty text
				User:      "U123456",
				Channel:   "C123456",
				TimeStamp: "1234567890.123456",
			},
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyCommandArgs(ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to command
	ctx.Custom["eventType"] = helpers.IncomingEventTypeCommand

	// Set up middleware args in context
	ctx.Custom["middlewareArgs"] = types.SlackCommandMiddlewareArgs{
		Command: types.SlashCommand{
			Command:   "/test",
			UserID:    "U123456",
			ChannelID: "C123456",
			Text:      "test parameters",
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyCommandArgsWithName(command string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to command
	ctx.Custom["eventType"] = helpers.IncomingEventTypeCommand

	// Set up middleware args in context
	ctx.Custom["middlewareArgs"] = types.SlackCommandMiddlewareArgs{
		Command: types.SlashCommand{
			Command:   command,
			UserID:    "U123456",
			ChannelID: "C123456",
			Text:      "test parameters",
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyBotMessageArgs(botID string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    "message",
		"subtype": "bot_message",
		"bot_id":  botID,
		"text":    "hi",
		"channel": "C123456",
		"ts":      "1234567890.123456",
	}
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: parsedEvent,
		Message: &types.MessageEvent{
			MessageEvent: slackevents.MessageEvent{
				SubType:   "bot_message",
				BotID:     botID,
				Text:      "hi",
				Channel:   "C123456",
				TimeStamp: "1234567890.123456",
			},
		},
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyReactionAddedArgs(userID string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":     "reaction_added",
		"user":     userID,
		"reaction": "thumbsup",
		"item": map[string]interface{}{
			"type":    "message",
			"channel": "C123456",
			"ts":      "1234567890.123456",
		},
	}
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: parsedEvent,
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyMemberChannelArgs(eventType, userID string, ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    eventType,
		"user":    userID,
		"channel": "C123456",
		"ts":      "1234567890.123456",
	}
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: parsedEvent,
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}

func createDummyAppHomeOpenedArgs(ctx *types.Context) types.AllMiddlewareArgs {
	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	// Set event type to event
	ctx.Custom["eventType"] = helpers.IncomingEventTypeEvent

	// Set up middleware args in context
	eventData := map[string]interface{}{
		"type":    "app_home_opened",
		"user":    "U123456",
		"channel": "D123456",
		"tab":     "home",
	}
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)

	ctx.Custom["middlewareArgs"] = types.SlackEventMiddlewareArgs{
		Event: parsedEvent,
	}

	return types.AllMiddlewareArgs{
		Context: ctx,
		Logger:  slog.Default(),
		Client:  &slack.Client{},
		Next: func() error {
			ctx.Custom["nextCalled"] = true
			return nil
		},
	}
}
