package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
)

func main() {
	// Get required environment variables
	appToken := os.Getenv("SLACK_APP_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	stateSecret := os.Getenv("SLACK_STATE_SECRET")

	if appToken == "" {
		log.Fatal("SLACK_APP_TOKEN environment variable is required")
	}
	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable is required")
	}

	// Initialize the app with Socket Mode and OAuth
	boltApp, err := app.New(app.AppOptions{
		LogLevel:      bolt.LogLevelDebug,
		SocketMode:    true,
		AppToken:      appToken,
		SigningSecret: signingSecret,
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		StateSecret:   stateSecret,
		Scopes:        []string{"channels:history", "chat:write", "commands"},
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Start Bolt App
	fmt.Println("⚡️ Bolt app is running! ⚡️")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Unable to start App: %v", err)
	}
}
