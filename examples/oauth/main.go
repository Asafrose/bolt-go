package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	bolt "github.com/Asafrose/bolt-go"
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

	// Initialize the app with OAuth configuration
	boltApp, err := app.New(app.AppOptions{
		LogLevel:          lo.ToPtr(bolt.LogLevelDebug),
		SigningSecret:     lo.ToPtr(signingSecret),
		ClientID:          lo.ToPtr(clientID),
		ClientSecret:      lo.ToPtr(clientSecret),
		StateSecret:       lo.ToPtr(stateSecret),
		Scopes:            []string{"chat:write"},
		InstallationStore: installationStore,
		InstallerOptions:  &types.InstallerOptions{
			// If this is true, /slack/install redirects installers to the Slack authorize URL
			// without rendering the web page with "Add to Slack" button.
			// This flag is available in @slack/bolt v3.7 or higher
			// DirectInstall: lo.ToPtr(true),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Start the app
	port := 3000
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			port = p
		}
	}

	fmt.Printf("⚡️ Bolt app started on port %d\n", port)
	fmt.Println("Visit http://localhost:3000/slack/install to install the app")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
