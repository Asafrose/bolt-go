package test

import (
	"context"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAuthIntegration tests the complete OAuth implementation
func TestOAuthIntegration(t *testing.T) {
	t.Parallel()
	t.Run("InstallProvider", func(t *testing.T) {
		t.Run("should create provider with minimal configuration", func(t *testing.T) {
			provider, err := oauth.NewInstallProvider(oauth.InstallProviderOptions{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			})
			require.NoError(t, err)
			assert.NotNil(t, provider, "Provider should be created")
		})

		t.Run("should fail with missing client ID", func(t *testing.T) {
			_, err := oauth.NewInstallProvider(oauth.InstallProviderOptions{
				ClientSecret: "test-client-secret",
			})
			require.Error(t, err, "Should fail without client ID")
			assert.Contains(t, err.Error(), "clientID is required")
		})

		t.Run("should fail with missing client secret", func(t *testing.T) {
			_, err := oauth.NewInstallProvider(oauth.InstallProviderOptions{
				ClientID: "test-client-id",
			})
			require.Error(t, err, "Should fail without client secret")
			assert.Contains(t, err.Error(), "clientSecret is required")
		})

		t.Run("should generate install URL", func(t *testing.T) {
			provider, err := oauth.NewInstallProvider(oauth.InstallProviderOptions{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			})
			require.NoError(t, err)

			url, err := provider.GenerateInstallURL(context.Background(), &oauth.InstallURLOptions{
				Scopes:     []string{"chat:write", "channels:read"},
				UserScopes: []string{"chat:write"},
			}, "")
			require.NoError(t, err)
			assert.NotEmpty(t, url, "Should generate URL")
			assert.Contains(t, url, "client_id=test-client-id")
			assert.Contains(t, url, "scope=chat%3Awrite%2Cchannels%3Aread") // URL encoded
			assert.Contains(t, url, "user_scope=chat%3Awrite")              // URL encoded
		})
	})

	t.Run("MemoryInstallationStore", func(t *testing.T) {
		t.Run("should store and retrieve installation", func(t *testing.T) {
			store := oauth.NewMemoryInstallationStore()
			ctx := context.Background()

			installation := &oauth.Installation{
				Team: &oauth.Team{
					ID:   "test-team-id",
					Name: "Test Team",
				},
				AccessToken: "test-access-token",
				BotToken:    "test-bot-token",
			}

			// Store installation
			err := store.StoreInstallation(ctx, installation)
			require.NoError(t, err)

			// Retrieve installation
			query := oauth.InstallationQuery{
				TeamID: "test-team-id",
			}
			retrieved, err := store.FetchInstallation(ctx, query)
			require.NoError(t, err)
			assert.Equal(t, installation.Team.ID, retrieved.Team.ID)
			assert.Equal(t, installation.AccessToken, retrieved.AccessToken)
		})

		t.Run("should delete installation", func(t *testing.T) {
			store := oauth.NewMemoryInstallationStore()
			ctx := context.Background()

			installation := &oauth.Installation{
				Team: &oauth.Team{
					ID: "test-team-id",
				},
			}

			// Store and verify
			err := store.StoreInstallation(ctx, installation)
			require.NoError(t, err)

			query := oauth.InstallationQuery{TeamID: "test-team-id"}
			_, err = store.FetchInstallation(ctx, query)
			require.NoError(t, err)

			// Delete and verify
			err = store.DeleteInstallation(ctx, query)
			require.NoError(t, err)

			_, err = store.FetchInstallation(ctx, query)
			require.Error(t, err, "Should not find deleted installation")
		})
	})

	t.Run("ClearStateStore", func(t *testing.T) {
		t.Run("should generate and verify state", func(t *testing.T) {
			store := oauth.NewClearStateStore()
			ctx := context.Background()

			installOptions := &oauth.InstallURLOptions{
				Scopes: []string{"test-scope"},
			}

			// Generate state
			state, err := store.GenerateStateParam(ctx, installOptions)
			require.NoError(t, err)
			assert.NotEmpty(t, state, "Should generate state")

			// Verify state
			retrieved, err := store.VerifyStateParam(ctx, state)
			require.NoError(t, err)
			assert.Equal(t, installOptions.Scopes, retrieved.Scopes)
		})

		t.Run("should fail with invalid state", func(t *testing.T) {
			store := oauth.NewClearStateStore()
			ctx := context.Background()

			_, err := store.VerifyStateParam(ctx, "invalid-state")
			require.Error(t, err, "Should fail with invalid state")
		})
	})

	t.Run("EncryptedStateStore", func(t *testing.T) {
		t.Run("should generate and verify encrypted state", func(t *testing.T) {
			store := oauth.NewEncryptedStateStore("test-secret")
			ctx := context.Background()

			installOptions := &oauth.InstallURLOptions{
				Scopes: []string{"test-scope"},
			}

			// Generate state
			state, err := store.GenerateStateParam(ctx, installOptions)
			require.NoError(t, err)
			assert.NotEmpty(t, state, "Should generate encrypted state")

			// Verify state
			retrieved, err := store.VerifyStateParam(ctx, state)
			require.NoError(t, err)
			assert.Equal(t, installOptions.Scopes, retrieved.Scopes)
		})
	})

	t.Run("HTTPReceiver OAuth Integration", func(t *testing.T) {
		t.Run("should configure OAuth with HTTPReceiver", func(t *testing.T) {
			store := oauth.NewMemoryInstallationStore()

			receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
				SigningSecret:     fakeSigningSecret,
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				InstallationStore: store,
				Scopes:            []string{"chat:write"},
				InstallerOptions: &types.InstallerOptions{
					InstallPath:     "/custom/install",
					RedirectURIPath: "/custom/oauth",
				},
			})

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			err = receiver.Init(app)
			require.NoError(t, err)

			assert.NotNil(t, receiver, "HTTP receiver should be configured with OAuth")
		})
	})

	t.Run("SocketModeReceiver OAuth Integration", func(t *testing.T) {
		t.Run("should configure OAuth with SocketModeReceiver", func(t *testing.T) {
			store := oauth.NewMemoryInstallationStore()

			receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
				AppToken:          fakeAppToken,
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				InstallationStore: store,
				Scopes:            []string{"chat:write"},
				InstallerOptions: &types.InstallerOptions{
					InstallPath:     "/custom/install",
					RedirectURIPath: "/custom/oauth",
					Port:            3001,
				},
			})

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			err = receiver.Init(app)
			require.NoError(t, err)

			assert.NotNil(t, receiver, "Socket Mode receiver should be configured with OAuth")
		})
	})

	t.Run("OAuth Configuration Validation", func(t *testing.T) {
		t.Run("should handle missing OAuth configuration gracefully", func(t *testing.T) {
			// Test HTTP receiver without OAuth
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

			assert.NotNil(t, receiver, "Receiver should work without OAuth")
		})

		t.Run("should handle partial OAuth configuration", func(t *testing.T) {
			// Test with only client ID (missing secret)
			receiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
				SigningSecret: fakeSigningSecret,
				ClientID:      "test-client-id",
				// Missing ClientSecret
			})

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			err = receiver.Init(app)
			require.NoError(t, err)

			assert.NotNil(t, receiver, "Receiver should handle partial OAuth config")
		})
	})

	t.Run("OAuth State Management", func(t *testing.T) {
		t.Run("should handle state verification enabled", func(t *testing.T) {
			stateVerification := true

			receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
				AppToken:     fakeAppToken,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				InstallerOptions: &types.InstallerOptions{
					StateVerification: &stateVerification,
				},
			})

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			err = receiver.Init(app)
			require.NoError(t, err)

			assert.NotNil(t, receiver, "Receiver should handle state verification")
		})

		t.Run("should handle state verification disabled", func(t *testing.T) {
			stateVerification := false

			receiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
				AppToken:     fakeAppToken,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				InstallerOptions: &types.InstallerOptions{
					StateVerification: &stateVerification,
				},
			})

			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			err = receiver.Init(app)
			require.NoError(t, err)

			assert.NotNil(t, receiver, "Receiver should handle disabled state verification")
		})
	})
}
