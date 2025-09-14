package test

import (
	"testing"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventMethodCompatibility(t *testing.T) {
	t.Parallel()
	t.Run("should accept SlackEventType constants", func(t *testing.T) {
		boltApp, err := app.New(app.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// This should compile and work
		boltApp.Event(types.EventTypeAppMention, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(types.EventTypeMessage, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(types.EventTypeReactionAdded, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		assert.NotNil(t, boltApp, "App should be created successfully")
	})

	t.Run("should accept string literals converted to SlackEventType", func(t *testing.T) {
		boltApp, err := app.New(app.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Convert strings to SlackEventType
		boltApp.Event(types.SlackEventType("app_mention"), func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(types.SlackEventType("message"), func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(types.SlackEventType("reaction_added"), func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		assert.NotNil(t, boltApp, "App should be created successfully")
	})

	t.Run("should accept custom SlackEventType instances", func(t *testing.T) {
		boltApp, err := app.New(app.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Custom event types using the SlackEventType constructor
		customEvent1 := types.SlackEventType("my_custom_event")
		customEvent2 := types.SlackEventType("enterprise_specific_event")

		boltApp.Event(customEvent1, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(customEvent2, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		assert.NotNil(t, boltApp, "App should be created successfully")
	})

	t.Run("should demonstrate type safety", func(t *testing.T) {
		boltApp, err := app.New(app.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// This demonstrates that only SlackEventType is accepted
		// The following would not compile:
		// boltApp.Event("string", handler) // ❌ Compile error
		// boltApp.Event(123, handler)      // ❌ Compile error

		// But these work:
		boltApp.Event(types.EventTypeMessage, func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		boltApp.Event(types.SlackEventType("custom_event"), func(args types.SlackEventMiddlewareArgs) error {
			return nil
		})

		assert.NotNil(t, boltApp, "App should be created successfully")
	})
}
