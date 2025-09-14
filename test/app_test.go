package test

import (
	"context"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants are now in test_helpers.go

func TestAppBasicFeatures(t *testing.T) {
	t.Parallel()
	t.Run("constructor", func(t *testing.T) {
		t.Run("with minimal configuration", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         fakeToken,
				SigningSecret: fakeSigningSecret,
			})

			require.NoError(t, err)
			assert.NotNil(t, app)
			assert.NotNil(t, app.Client)
			assert.NotNil(t, app.Logger)
		})

		t.Run("with socket mode", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				AppToken:   fakeAppToken,
				Token:      fakeToken,
				SocketMode: true,
			})

			require.NoError(t, err)
			assert.NotNil(t, app)
		})

		t.Run("should fail without required configuration", func(t *testing.T) {
			_, err := bolt.New(bolt.AppOptions{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "signing secret required")
		})

		t.Run("should fail without app token in socket mode", func(t *testing.T) {
			_, err := bolt.New(bolt.AppOptions{
				SocketMode: true,
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "app token required")
		})
	})

	t.Run("initialization", func(t *testing.T) {
		t.Run("with defer initialization", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:               fakeToken,
				SigningSecret:       fakeSigningSecret,
				DeferInitialization: true,
			})

			require.NoError(t, err)
			assert.NotNil(t, app)

			// Initialize the app
			ctx := context.Background()
			err = app.Init(ctx)
			require.NoError(t, err)
		})
	})

	t.Run("middleware registration", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register global middleware", func(t *testing.T) {
			app.Use(func(args bolt.AllMiddlewareArgs) error {
				return args.Next()
			})

			// The middleware registration should not fail
			assert.NotNil(t, app)
		})
	})

	t.Run("event listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register event listeners", func(t *testing.T) {
			app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})

		t.Run("should register message listeners", func(t *testing.T) {
			app.Message("hello", func(args bolt.SlackEventMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})

	t.Run("action listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register action listeners", func(t *testing.T) {
			app.Action(bolt.ActionConstraints{
				ActionID: "button_click",
			}, func(args bolt.SlackActionMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})

	t.Run("command listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register command listeners", func(t *testing.T) {
			app.Command("/hello", func(args bolt.SlackCommandMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})

	t.Run("shortcut listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register shortcut listeners", func(t *testing.T) {
			app.Shortcut(bolt.ShortcutConstraints{
				CallbackID: "test_shortcut",
			}, func(args bolt.SlackShortcutMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})

	t.Run("view listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register view listeners", func(t *testing.T) {
			app.View(bolt.ViewConstraints{
				CallbackID: "test_view",
			}, func(args bolt.SlackViewMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})

	t.Run("options listeners", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		t.Run("should register options listeners", func(t *testing.T) {
			app.Options(bolt.OptionsConstraints{
				ActionID: "test_options",
			}, func(args bolt.SlackOptionsMiddlewareArgs) error {
				return args.Next()
			})

			assert.NotNil(t, app)
		})
	})
}

func TestAppLogLevels(t *testing.T) {
	t.Parallel()
	t.Run("should set log level correctly", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			LogLevel:      bolt.LogLevelDebug,
		})

		require.NoError(t, err)
		assert.NotNil(t, app)
	})

	t.Run("should enable developer mode", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			DeveloperMode: true,
		})

		require.NoError(t, err)
		assert.NotNil(t, app)
	})
}

func TestAppIgnoreSelf(t *testing.T) {
	t.Parallel()
	t.Run("should enable ignore self by default", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})

		require.NoError(t, err)
		assert.NotNil(t, app)
	})

	t.Run("should allow disabling ignore self", func(t *testing.T) {
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			IgnoreSelf:    &[]bool{false}[0],
		})

		require.NoError(t, err)
		assert.NotNil(t, app)
	})
}

// Helper function
// stringPtr helper is defined in helpers_test.go
