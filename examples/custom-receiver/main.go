package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/samber/lo"
)

func main() {
	// Get required environment variables
	token := os.Getenv("SLACK_BOT_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")

	if token == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable is required")
	}
	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable is required")
	}

	// Initialize the app with default HTTP receiver
	boltApp, err := app.New(app.AppOptions{
		Token:         lo.ToPtr(token),
		SigningSecret: lo.ToPtr(signingSecret),
		LogLevel:      lo.ToPtr(app.LogLevelDebug),
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Start your app
	fmt.Println("⚡️ Bolt app with default receiver is running!")
	fmt.Println("Check the gin-receiver/ and echo-receiver/ subdirectories for custom receiver implementations")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
