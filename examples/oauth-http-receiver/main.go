package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/samber/lo"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/types"
)

func main() {
	// Get required environment variables
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	stateSecret := os.Getenv("SLACK_STATE_SECRET")

	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable is required")
	}
	if clientID == "" {
		log.Fatal("SLACK_CLIENT_ID environment variable is required")
	}
	if clientSecret == "" {
		log.Fatal("SLACK_CLIENT_SECRET environment variable is required")
	}

	// Create installation store
	installationStore := oauth.NewMemoryInstallationStore()

	// Create the Bolt App with HTTP receiver and OAuth support
	boltApp, err := app.New(app.AppOptions{
		SigningSecret:     lo.ToPtr(signingSecret),
		ClientID:          lo.ToPtr(clientID),
		ClientSecret:      lo.ToPtr(clientSecret),
		StateSecret:       lo.ToPtr(stateSecret),
		Scopes:            []string{"chat:write"},
		InstallationStore: installationStore,
		LogLevel:          lo.ToPtr(app.LogLevelDebug), // set loglevel at the App level
		InstallerOptions:  &types.InstallerOptions{
			// DirectInstall: lo.ToPtr(true),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Slack interactions like listening for events are methods on app
	boltApp.Event("message", func(args types.SlackEventMiddlewareArgs) error {
		// Do some slack-specific stuff here
		// You can use the client to make API calls
		args.Logger.Info("Received message event", "event", args.Event)
		return nil
	})

	// Start the app
	fmt.Println("⚡️ HTTP app with OAuth is running")
	fmt.Println("Visit http://localhost:3000/slack/install to install the app")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
