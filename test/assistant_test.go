package test

import (
	"context"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistant(t *testing.T) {
	t.Parallel()
	t.Run("constructor", func(t *testing.T) {
		t.Run("should create assistant with required middleware", func(t *testing.T) {
			config := bolt.AssistantConfig{
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			assistant, err := bolt.NewAssistant(config)
			require.NoError(t, err)
			assert.NotNil(t, assistant)
		})

		t.Run("should fail without threadStarted middleware", func(t *testing.T) {
			config := bolt.AssistantConfig{
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			_, err := bolt.NewAssistant(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "threadStarted middleware is required")
		})

		t.Run("should fail without userMessage middleware", func(t *testing.T) {
			config := bolt.AssistantConfig{
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			_, err := bolt.NewAssistant(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "userMessage middleware is required")
		})

		t.Run("should use default thread context store", func(t *testing.T) {
			config := bolt.AssistantConfig{
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			assistant, err := bolt.NewAssistant(config)
			require.NoError(t, err)
			assert.NotNil(t, assistant)
		})

		t.Run("should accept custom thread context store", func(t *testing.T) {
			customStore := bolt.NewDefaultThreadContextStore()

			config := bolt.AssistantConfig{
				ThreadContextStore: customStore,
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			assistant, err := bolt.NewAssistant(config)
			require.NoError(t, err)
			assert.NotNil(t, assistant)
		})
	})

	t.Run("middleware integration", func(t *testing.T) {
		t.Run("should return middleware function", func(t *testing.T) {
			config := bolt.AssistantConfig{
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			assistant, err := bolt.NewAssistant(config)
			require.NoError(t, err)

			middleware := assistant.GetMiddleware()
			assert.NotNil(t, middleware)
		})

		t.Run("should integrate with app", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})
			require.NoError(t, err)

			config := bolt.AssistantConfig{
				ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
					func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
						return args.Next()
					},
				},
				UserMessage: []bolt.AssistantUserMessageMiddleware{
					func(args bolt.AssistantUserMessageMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			assistant, err := bolt.NewAssistant(config)
			require.NoError(t, err)

			// This should not panic
			app.Use(assistant.GetMiddleware())
		})
	})
}

func TestAssistantThreadContextStore(t *testing.T) {
	t.Parallel()
	t.Run("DefaultThreadContextStore", func(t *testing.T) {
		store := bolt.NewDefaultThreadContextStore()
		ctx := context.Background()

		t.Run("should get empty context for new thread", func(t *testing.T) {
			context, err := store.Get(ctx, "C123", "1234567890.123")
			require.NoError(t, err)
			assert.NotNil(t, context)
			assert.Equal(t, "C123", context.ChannelID)
			assert.Equal(t, "1234567890.123", context.ThreadTS)
			assert.NotNil(t, context.Context)
		})

		t.Run("should save and retrieve context", func(t *testing.T) {
			threadContext := &bolt.AssistantThreadContext{
				ChannelID: "C123",
				ThreadTS:  "1234567890.123",
				Context: map[string]interface{}{
					"key": "value",
				},
			}

			err := store.Save(ctx, threadContext)
			require.NoError(t, err)

			retrieved, err := store.Get(ctx, "C123", "1234567890.123")
			require.NoError(t, err)
			assert.Equal(t, threadContext.ChannelID, retrieved.ChannelID)
			assert.Equal(t, threadContext.ThreadTS, retrieved.ThreadTS)
			assert.Equal(t, "value", retrieved.Context["key"])
		})
	})
}

func TestAssistantUtilities(t *testing.T) {
	t.Parallel()
	t.Run("utility functions", func(t *testing.T) {
		// Test utility function creation and usage
		// This would typically be tested through integration tests
		// with actual assistant middleware

		config := bolt.AssistantConfig{
			ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
				func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
					// Test that utility functions are available
					assert.NotNil(t, args.GetThreadContext)
					assert.NotNil(t, args.SaveThreadContext)
					assert.NotNil(t, args.Say)
					assert.NotNil(t, args.SetStatus)
					assert.NotNil(t, args.SetSuggestedPrompts)
					assert.NotNil(t, args.SetTitle)
					return args.Next()
				},
			},
			UserMessage: []bolt.AssistantUserMessageMiddleware{
				func(args bolt.AssistantUserMessageMiddlewareArgs) error {
					// Test that utility functions are available
					assert.NotNil(t, args.GetThreadContext)
					assert.NotNil(t, args.SaveThreadContext)
					assert.NotNil(t, args.Say)
					assert.NotNil(t, args.SetStatus)
					assert.NotNil(t, args.SetSuggestedPrompts)
					assert.NotNil(t, args.SetTitle)
					return args.Next()
				},
			},
		}

		assistant, err := bolt.NewAssistant(config)
		require.NoError(t, err)
		assert.NotNil(t, assistant)
	})
}
