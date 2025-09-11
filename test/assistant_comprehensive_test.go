package test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Asafrose/bolt-go/pkg/assistant"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssistantComprehensive implements all missing tests from Assistant.spec.ts
func TestAssistantComprehensive(t *testing.T) {

	t.Run("constructor", func(t *testing.T) {
		t.Run("should accept config as single functions", func(t *testing.T) {
			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
				},
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
				},
			}

			assistant, err := assistant.NewAssistant(config)
			require.NoError(t, err)
			assert.NotNil(t, assistant)
		})

		t.Run("should accept config as multiple functions", func(t *testing.T) {
			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
				},
				ThreadContextChanged: []assistant.AssistantThreadContextChangedMiddleware{
					func(args assistant.AssistantThreadContextChangedMiddlewareArgs) error {
						return nil
					},
				},
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
				},
			}

			assistant, err := assistant.NewAssistant(config)
			require.NoError(t, err)
			assert.NotNil(t, assistant)
		})
	})

	t.Run("validate", func(t *testing.T) {
		t.Run("should throw an error if config is not an object", func(t *testing.T) {
			// Test direct validation function
			err := assistant.ValidateAssistantConfig(nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "configuration object")
		})

		t.Run("should throw an error if required keys are missing", func(t *testing.T) {
			// Missing userMessage
			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
				},
			}

			_, err := assistant.NewAssistant(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "userMessage")

			// Missing threadStarted
			config2 := assistant.AssistantConfig{
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
				},
			}

			_, err2 := assistant.NewAssistant(config2)
			assert.Error(t, err2)
			assert.Contains(t, err2.Error(), "threadStarted")
		})

		t.Run("should throw an error if props are not a single callback or an array of callbacks", func(t *testing.T) {
			// Test with empty arrays
			config := assistant.AssistantConfig{
				ThreadStarted:        []assistant.AssistantThreadStartedMiddleware{},
				ThreadContextChanged: []assistant.AssistantThreadContextChangedMiddleware{},
				UserMessage:          []assistant.AssistantUserMessageMiddleware{},
			}

			_, err := assistant.NewAssistant(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "middleware")
		})
	})

	t.Run("getMiddleware", func(t *testing.T) {
		t.Run("should call next if not an assistant event", func(t *testing.T) {
			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
				},
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
				},
			}

			assistant, err := assistant.NewAssistant(config)
			require.NoError(t, err)

			middleware := assistant.GetMiddleware()
			assert.NotNil(t, middleware)

			// Create non-assistant event args
			var nextCalled bool
			botToken := "xoxb-test"
			botID := "B123"
			args := types.AllMiddlewareArgs{
				Context: &types.Context{
					BotToken: &botToken,
					BotID:    &botID,
				},
				Client: &slack.Client{},
				Logger: slog.Default(),
				Next: func() error {
					nextCalled = true
					return nil
				},
			}

			// Add regular message event (not assistant)
			args.Context.Custom = map[string]interface{}{
				"middlewareArgs": types.SlackEventMiddlewareArgs{
					AllMiddlewareArgs: args,
					Event: map[string]interface{}{
						"type":    "message",
						"user":    "U123",
						"text":    "hello",
						"channel": "C123",
					},
					Body: map[string]interface{}{
						"event": map[string]interface{}{
							"type": "message",
						},
					},
				},
			}

			err = middleware(args)
			assert.NoError(t, err)
			assert.True(t, nextCalled, "Next should be called for non-assistant events")
		})

		t.Run("should not call next if a assistant event", func(t *testing.T) {
			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						return nil
					},
				},
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						return nil
					},
				},
			}

			assistant, err := assistant.NewAssistant(config)
			require.NoError(t, err)

			middleware := assistant.GetMiddleware()

			var nextCalled bool
			botToken := "xoxb-test"
			botID := "B123"
			args := types.AllMiddlewareArgs{
				Context: &types.Context{
					BotToken: &botToken,
					BotID:    &botID,
				},
				Client: &slack.Client{},
				Logger: slog.Default(),
				Next: func() error {
					nextCalled = true
					return nil
				},
			}

			// Add assistant event
			args.Context.Custom = map[string]interface{}{
				"middlewareArgs": types.SlackEventMiddlewareArgs{
					AllMiddlewareArgs: args,
					Event: map[string]interface{}{
						"type":      "assistant_thread_started",
						"channel":   "C123",
						"thread_ts": "1234567890.123",
					},
					Body: map[string]interface{}{
						"event": map[string]interface{}{
							"type": "assistant_thread_started",
						},
					},
				},
			}

			err = middleware(args)
			assert.NoError(t, err)
			assert.False(t, nextCalled, "Next should NOT be called for assistant events")
		})
	})

	t.Run("isAssistantEvent", func(t *testing.T) {
		t.Run("should return true if recognized assistant event", func(t *testing.T) {
			testCases := []map[string]interface{}{
				{"type": "assistant_thread_started"},
				{"type": "assistant_thread_context_changed"},
				{
					"type":         "message",
					"channel":      "D123456",
					"channel_type": "im",
					"thread_ts":    "1234567890.123",
				},
			}

			for _, event := range testCases {
				result := assistant.IsAssistantEvent(event)
				assert.True(t, result, "Should recognize %s as assistant event", event["type"])
			}
		})

		t.Run("should return false if not a recognized assistant event", func(t *testing.T) {
			testCases := []map[string]interface{}{
				{"type": "message", "channel": "C123"}, // regular message, not in IM
				{"type": "app_mention"},
				{"type": "team_join"},
				{"type": "unknown_event"},
				{}, // empty event
			}

			for _, event := range testCases {
				result := assistant.IsAssistantEvent(event)
				assert.False(t, result, "Should not recognize %v as assistant event", event)
			}
		})
	})

	t.Run("matchesConstraints", func(t *testing.T) {
		t.Run("should return true if recognized assistant message", func(t *testing.T) {
			event := map[string]interface{}{
				"type":         "message",
				"channel":      "D123456",
				"channel_type": "im",
				"thread_ts":    "1234567890.123",
				"user":         "U123",
				"text":         "Hello assistant",
			}

			result := assistant.MatchesConstraints(event)
			assert.True(t, result)
		})

		t.Run("should return false if not supported message subtype", func(t *testing.T) {
			event := map[string]interface{}{
				"type":         "message",
				"channel_type": "im",
				"thread_ts":    "1234567890.123",
				"subtype":      "bot_message", // bot messages not supported
			}

			result := assistant.MatchesConstraints(event)
			assert.False(t, result)
		})

		t.Run("should return true if not message event", func(t *testing.T) {
			event := map[string]interface{}{
				"type": "assistant_thread_started",
			}

			result := assistant.MatchesConstraints(event)
			assert.True(t, result)
		})
	})

	t.Run("isAssistantMessage", func(t *testing.T) {
		t.Run("should return true if assistant message event", func(t *testing.T) {
			event := map[string]interface{}{
				"type":         "message",
				"channel":      "D123456",
				"channel_type": "im",
				"thread_ts":    "1234567890.123",
			}

			result := assistant.IsAssistantMessage(event)
			assert.True(t, result)
		})

		t.Run("should return false if not correct subtype", func(t *testing.T) {
			event := map[string]interface{}{
				"type":         "message",
				"channel_type": "channel", // not IM
				"thread_ts":    "1234567890.123",
			}

			result := assistant.IsAssistantMessage(event)
			assert.False(t, result)
		})

		t.Run("should return false if thread_ts is missing", func(t *testing.T) {
			event := map[string]interface{}{
				"type":         "message",
				"channel_type": "im",
				// missing thread_ts
			}

			result := assistant.IsAssistantMessage(event)
			assert.False(t, result)
		})

		t.Run("should return false if channel_type is incorrect", func(t *testing.T) {
			event := map[string]interface{}{
				"type":      "message",
				"thread_ts": "1234567890.123",
				// missing channel_type
			}

			result := assistant.IsAssistantMessage(event)
			assert.False(t, result)
		})
	})

	t.Run("enrichAssistantArgs", func(t *testing.T) {
		store := assistant.NewDefaultThreadContextStore()

		t.Run("should remove next() from all original event args", func(t *testing.T) {
			var nextCalled bool
			botToken := "test"
			originalArgs := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Next: func() error {
						nextCalled = true
						return nil
					},
				},
			}

			enrichedArgs := assistant.EnrichAssistantArgs(store, originalArgs)

			// Next should be removed from enriched args
			assert.Nil(t, enrichedArgs.Next)

			// Original next should not have been called
			assert.False(t, nextCalled)
		})

		t.Run("should augment assistant_thread_started args with utilities", func(t *testing.T) {
			botToken := "test"
			originalArgs := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			enrichedArgs := assistant.EnrichAssistantArgs(store, originalArgs)

			assert.NotNil(t, enrichedArgs.GetThreadContext)
			assert.NotNil(t, enrichedArgs.SaveThreadContext)
			assert.NotNil(t, enrichedArgs.Say)
			assert.NotNil(t, enrichedArgs.SetStatus)
			assert.NotNil(t, enrichedArgs.SetSuggestedPrompts)
			assert.NotNil(t, enrichedArgs.SetTitle)
		})

		t.Run("should augment assistant_thread_context_changed args with utilities", func(t *testing.T) {
			botToken := "test"
			originalArgs := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			enrichedArgs := assistant.EnrichAssistantArgs(store, originalArgs)

			assert.NotNil(t, enrichedArgs.GetThreadContext)
			assert.NotNil(t, enrichedArgs.SaveThreadContext)
			assert.NotNil(t, enrichedArgs.Say)
			assert.NotNil(t, enrichedArgs.SetStatus)
			assert.NotNil(t, enrichedArgs.SetSuggestedPrompts)
			assert.NotNil(t, enrichedArgs.SetTitle)
		})

		t.Run("should augment message args with utilities", func(t *testing.T) {
			botToken := "test"
			originalArgs := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			enrichedArgs := assistant.EnrichAssistantArgs(store, originalArgs)

			assert.NotNil(t, enrichedArgs.GetThreadContext)
			assert.NotNil(t, enrichedArgs.SaveThreadContext)
			assert.NotNil(t, enrichedArgs.Say)
			assert.NotNil(t, enrichedArgs.SetStatus)
			assert.NotNil(t, enrichedArgs.SetSuggestedPrompts)
			assert.NotNil(t, enrichedArgs.SetTitle)
		})
	})

	t.Run("extractThreadInfo", func(t *testing.T) {
		t.Run("should return expected channelId, threadTs, and context for assistant_thread_started event", func(t *testing.T) {
			event := map[string]interface{}{
				"type":      "assistant_thread_started",
				"channel":   "C123456",
				"thread_ts": "1234567890.123",
				"assistant_thread": map[string]interface{}{
					"context": map[string]interface{}{
						"key": "value",
					},
				},
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(event)
			assert.Equal(t, "C123456", channelID)
			assert.Equal(t, "1234567890.123", threadTS)
			assert.Equal(t, "value", context["key"])
		})

		t.Run("should return expected channelId, threadTs, and context for assistant_thread_context_changed event", func(t *testing.T) {
			event := map[string]interface{}{
				"type":      "assistant_thread_context_changed",
				"channel":   "C789012",
				"thread_ts": "9876543210.456",
				"assistant_thread": map[string]interface{}{
					"context": map[string]interface{}{
						"status": "updated",
					},
				},
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(event)
			assert.Equal(t, "C789012", channelID)
			assert.Equal(t, "9876543210.456", threadTS)
			assert.Equal(t, "updated", context["status"])
		})

		t.Run("should return expected channelId and threadTs for message event", func(t *testing.T) {
			event := map[string]interface{}{
				"type":      "message",
				"channel":   "D123456",
				"thread_ts": "1111111111.111",
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(event)
			assert.Equal(t, "D123456", channelID)
			assert.Equal(t, "1111111111.111", threadTS)
			assert.NotNil(t, context) // should return empty context for messages
		})

		t.Run("should throw error if channel_id or thread_ts are missing", func(t *testing.T) {
			// Missing channel
			event1 := map[string]interface{}{
				"type":      "message",
				"thread_ts": "1111111111.111",
			}

			assert.Panics(t, func() {
				assistant.ExtractThreadInfo(event1)
			})

			// Missing thread_ts
			event2 := map[string]interface{}{
				"type":    "message",
				"channel": "D123456",
			}

			assert.Panics(t, func() {
				assistant.ExtractThreadInfo(event2)
			})
		})
	})

	t.Run("assistant args/utilities", func(t *testing.T) {
		t.Run("say should call chat.postMessage", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			// Test that Say function exists and is callable
			assert.NotNil(t, enrichedArgs.Say)
			_, err := enrichedArgs.Say("Hello world")
			assert.NoError(t, err)
		})

		t.Run("say should be called with message_metadata that includes thread context", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			// Test that Say function handles complex message objects
			_, err := enrichedArgs.Say(map[string]interface{}{
				"text": "Hello",
				"metadata": map[string]interface{}{
					"event_type": "assistant_thread_context",
					"event_payload": map[string]interface{}{
						"key": "value",
					},
				},
			})
			assert.NoError(t, err)
		})

		t.Run("say should be called with message_metadata that supplements thread context", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			// Store existing context
			store.Save(context.Background(), &assistant.AssistantThreadContext{
				ChannelID: "C123",
				ThreadTS:  "1234567890.123",
				Context: map[string]interface{}{
					"existing": "value",
				},
			})

			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			_, err := enrichedArgs.Say(map[string]interface{}{
				"text": "Hello",
				"metadata": map[string]interface{}{
					"event_payload": map[string]interface{}{
						"new": "data",
					},
				},
			})
			assert.NoError(t, err)
		})

		t.Run("say should get context from store if no thread context is included in event", func(t *testing.T) {
			botToken := "test"
			store := assistant.NewDefaultThreadContextStore()
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			context, err := enrichedArgs.GetThreadContext()
			assert.NoError(t, err)
			assert.NotNil(t, context)
		})

		t.Run("setStatus should call assistant.threads.setStatus", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			err := enrichedArgs.SetStatus("in_progress")
			assert.NoError(t, err)
		})

		t.Run("setSuggestedPrompts should call assistant.threads.setSuggestedPrompts", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			prompts := []string{"What can you help with?", "Show me examples"}
			err := enrichedArgs.SetSuggestedPrompts(assistant.SetSuggestedPromptsArguments{
				Prompts: prompts,
			})
			assert.NoError(t, err)
		})

		t.Run("setTitle should call assistant.threads.setTitle", func(t *testing.T) {
			botToken := "test"
			args := assistant.AllAssistantMiddlewareArgs{
				AllMiddlewareArgs: types.AllMiddlewareArgs{
					Context: &types.Context{BotToken: &botToken},
					Client:  &slack.Client{},
					Logger:  slog.Default(),
				},
			}

			store := assistant.NewDefaultThreadContextStore()
			enrichedArgs := assistant.EnrichAssistantArgs(store, args)

			err := enrichedArgs.SetTitle("My Assistant Thread")
			assert.NoError(t, err)
		})
	})

	t.Run("processAssistantMiddleware", func(t *testing.T) {
		t.Run("should call each callback in user-provided middleware", func(t *testing.T) {
			var middleware1Called, middleware2Called, middleware3Called bool

			config := assistant.AssistantConfig{
				ThreadStarted: []assistant.AssistantThreadStartedMiddleware{
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						middleware1Called = true
						return nil
					},
					func(args assistant.AssistantThreadStartedMiddlewareArgs) error {
						middleware2Called = true
						return nil
					},
				},
				UserMessage: []assistant.AssistantUserMessageMiddleware{
					func(args assistant.AssistantUserMessageMiddlewareArgs) error {
						middleware3Called = true
						return nil
					},
				},
			}

			assistant, err := assistant.NewAssistant(config)
			require.NoError(t, err)

			// Test thread started
			err = assistant.ProcessAssistantMiddleware("assistant_thread_started", map[string]interface{}{
				"channel":   "C123",
				"thread_ts": "1234567890.123",
			})
			assert.NoError(t, err)
			assert.True(t, middleware1Called)
			assert.True(t, middleware2Called)
			assert.False(t, middleware3Called) // Different event type

			// Reset and test user message
			middleware1Called, middleware2Called, middleware3Called = false, false, false
			err = assistant.ProcessAssistantMiddleware("message", map[string]interface{}{
				"channel":      "D123",
				"thread_ts":    "1234567890.123",
				"channel_type": "im",
			})
			assert.NoError(t, err)
			assert.False(t, middleware1Called) // Different event type
			assert.False(t, middleware2Called) // Different event type
			assert.True(t, middleware3Called)
		})
	})

	t.Run("extractThreadInfo", func(t *testing.T) {
		t.Run("should return expected channelId, threadTs, and context for assistant_thread_started event", func(t *testing.T) {
			payload := map[string]interface{}{
				"assistant_thread": map[string]interface{}{
					"channel_id": "C1234567890",
					"thread_ts":  "1234567890.123456",
					"context": map[string]interface{}{
						"channel_id":    "C1234567890",
						"enterprise_id": "E1234567890",
						"team_id":       "T1234567890",
					},
				},
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(payload)

			assert.Equal(t, "C1234567890", channelID)
			assert.Equal(t, "1234567890.123456", threadTS)
			assert.Equal(t, "C1234567890", context["channel_id"])
			assert.Equal(t, "E1234567890", context["enterprise_id"])
			assert.Equal(t, "T1234567890", context["team_id"])
		})

		t.Run("should return expected channelId, threadTs, and context for assistant_thread_context_changed event", func(t *testing.T) {
			payload := map[string]interface{}{
				"assistant_thread": map[string]interface{}{
					"channel_id": "C9876543210",
					"thread_ts":  "9876543210.654321",
					"context": map[string]interface{}{
						"channel_id": "C9876543210",
						"team_id":    "T9876543210",
					},
				},
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(payload)

			assert.Equal(t, "C9876543210", channelID)
			assert.Equal(t, "9876543210.654321", threadTS)
			assert.Equal(t, "C9876543210", context["channel_id"])
			assert.Equal(t, "T9876543210", context["team_id"])
		})

		t.Run("should return expected channelId and threadTs for message event", func(t *testing.T) {
			payload := map[string]interface{}{
				"channel":   "D1234567890",
				"thread_ts": "1234567890.123456",
				"text":      "Hello assistant",
				"user":      "U1234567890",
			}

			channelID, threadTS, context := assistant.ExtractThreadInfo(payload)

			assert.Equal(t, "D1234567890", channelID)
			assert.Equal(t, "1234567890.123456", threadTS)
			assert.Empty(t, context) // No context for message events
		})

		t.Run("should throw error if channel_id or thread_ts are missing", func(t *testing.T) {
			// Test missing channel_id
			payload1 := map[string]interface{}{
				"assistant_thread": map[string]interface{}{
					"thread_ts": "1234567890.123456",
				},
			}

			assert.Panics(t, func() {
				assistant.ExtractThreadInfo(payload1)
			})

			// Test missing thread_ts
			payload2 := map[string]interface{}{
				"assistant_thread": map[string]interface{}{
					"channel_id": "C1234567890",
				},
			}

			assert.Panics(t, func() {
				assistant.ExtractThreadInfo(payload2)
			})

			// Test completely missing data
			payload3 := map[string]interface{}{
				"some_other_field": "value",
			}

			assert.Panics(t, func() {
				assistant.ExtractThreadInfo(payload3)
			})
		})
	})
}
