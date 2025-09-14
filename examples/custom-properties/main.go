package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/samber/lo"
	"github.com/Asafrose/bolt-go/pkg/types"
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

	// Initialize the app
	boltApp, err := app.New(app.AppOptions{
		Token:         lo.ToPtr(token),
		SigningSecret: lo.ToPtr(signingSecret),
		LogLevel:      lo.ToPtr(bolt.LogLevelDebug),
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Add middleware that logs the context (including custom properties)
	boltApp.Use(func(args types.AllMiddlewareArgs) error {
		args.Logger.Info("Request context", "context", args.Context)
		return args.Next()
	})

	// Start your app
	fmt.Println("⚡️ Bolt app with custom properties is running!")
	fmt.Println("Check the http/ and socket-mode/ subdirectories for specific implementations")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
