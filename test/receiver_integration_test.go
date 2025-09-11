package test

import (
	"context"
	"testing"
	"time"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPReceiverIntegration(t *testing.T) {
	t.Parallel()
	t.Run("should create HTTP receiver with valid options", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		assert.NotNil(t, receiver, "HTTP receiver should be created")
	})

	t.Run("should initialize with app", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err, "Receiver should initialize with app")
	})

	t.Run("should start and stop receiver lifecycle", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Test that we can start the receiver (will bind to a port)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Start in a goroutine since it's blocking
		startErr := make(chan error, 1)
		go func() {
			startErr <- receiver.Start(ctx)
		}()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Stop the receiver
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer stopCancel()

		err = receiver.Stop(stopCtx)
		require.NoError(t, err, "Receiver should stop without error")

		// Wait for start to complete
		select {
		case err := <-startErr:
			// Context cancellation is expected
			require.Error(t, err, "Start should return error when context is cancelled")
		case <-time.After(2 * time.Second):
			t.Error("Start did not return within timeout")
		}
	})

	t.Run("should handle custom endpoints", func(t *testing.T) {
		customEndpoints := &types.ReceiverEndpoints{
			Events:      "/custom/events",
			Commands:    "/custom/commands",
			Interactive: "/custom/interactive",
		}

		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
			Endpoints:     customEndpoints,
		})

		assert.NotNil(t, receiver, "HTTP receiver should be created with custom endpoints")

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err, "Receiver should initialize with custom endpoints")
	})

	t.Run("should handle process before response option", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret:         fakeSigningSecret,
			ProcessBeforeResponse: true,
		})

		assert.NotNil(t, receiver, "HTTP receiver should be created with process before response")

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err, "Receiver should initialize with process before response")
	})
}

func TestSocketModeReceiverIntegration(t *testing.T) {
	t.Parallel()
	t.Run("should create socket mode receiver with valid options", func(t *testing.T) {
		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken: fakeAppToken,
		})

		assert.NotNil(t, receiver, "Socket mode receiver should be created")
	})

	t.Run("should initialize with app", func(t *testing.T) {
		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken: fakeAppToken,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err, "Socket mode receiver should initialize with app")
	})

	t.Run("should handle connection lifecycle", func(t *testing.T) {
		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken: fakeAppToken,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Test that we can start the receiver - with the new socketmode client,
		// Start() doesn't return connection errors immediately since connection happens in background
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start in a goroutine since it's blocking
		startErr := make(chan error, 1)
		go func() {
			startErr <- receiver.Start(ctx)
		}()

		// Wait for start to complete or timeout
		select {
		case err := <-startErr:
			// Start should complete successfully even with fake token since connection is in background
			require.NoError(t, err, "Start should return without error even with fake token")
		case <-time.After(200 * time.Millisecond):
			t.Error("Start did not return within timeout")
		}
	})

	t.Run("should handle custom properties", func(t *testing.T) {
		customProps := map[string]interface{}{
			"custom_key":   "custom_value",
			"ping_timeout": 30,
		}

		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken:         fakeAppToken,
			CustomProperties: customProps,
		})

		assert.NotNil(t, receiver, "Socket mode receiver should be created with custom properties")

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err, "Receiver should initialize with custom properties")
	})
}

func TestReceiverOptions(t *testing.T) {
	t.Parallel()
	t.Run("should validate HTTP receiver options", func(t *testing.T) {
		// Test with minimal options
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})
		assert.NotNil(t, receiver)

		// Test with all options
		receiver = receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
			Endpoints: &types.ReceiverEndpoints{
				Events:      "/custom/events",
				Commands:    "/custom/commands",
				Interactive: "/custom/interactive",
			},
			ProcessBeforeResponse:         true,
			UnhandledRequestTimeoutMillis: 5000,
			CustomProperties: map[string]interface{}{
				"custom_key": "custom_value",
			},
		})
		assert.NotNil(t, receiver)
	})

	t.Run("should validate socket mode receiver options", func(t *testing.T) {
		// Test with minimal options
		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken: fakeAppToken,
		})
		assert.NotNil(t, receiver)

		// Test with all options
		receiver = receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken:    fakeAppToken,
			PingTimeout: 30,
			CustomProperties: map[string]interface{}{
				"custom_key": "custom_value",
			},
		})
		assert.NotNil(t, receiver)
	})
}

func TestReceiverErrorHandling(t *testing.T) {
	t.Parallel()
	t.Run("should handle initialization without app", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Starting without initialization should fail
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := receiver.Start(ctx)
		require.Error(t, err, "Should fail to start without initialization")
	})

	t.Run("should handle socket mode initialization without app", func(t *testing.T) {
		receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
			AppToken: fakeAppToken,
		})

		// With the new socketmode client, starting without app initialization doesn't fail immediately
		// The app field is only used when processing events, not during connection setup
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := receiver.Start(ctx)
		require.NoError(t, err, "Start should succeed even without app initialization")
	})

	t.Run("should handle stop before start", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Stop before start should not panic
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = receiver.Stop(ctx)
		// This might return an error or nil depending on implementation, but shouldn't panic
		// We just verify it doesn't crash
		_ = err
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create a context that gets cancelled immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Start should handle the cancelled context gracefully
		err = receiver.Start(ctx)
		require.Error(t, err, "Should return error for cancelled context")
		assert.Contains(t, err.Error(), "context", "Error should mention context cancellation")
	})
}
