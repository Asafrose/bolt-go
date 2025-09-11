package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
)

func main() {
	// Get required environment variables
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable is required")
	}

	// Create HTTP receiver with custom properties extractor
	httpReceiver := receivers.NewHTTPReceiver(types.HTTPReceiverOptions{
		SigningSecret: signingSecret,
		CustomProperties: map[string]interface{}{
			"example_property": "custom_value",
		},
		// Note: Custom properties extractor would need to be implemented
		// in the actual receiver to extract properties from HTTP requests
	})

	// Initialize the app with custom receiver
	boltApp, err := app.New(app.AppOptions{
		Token:    &[]string{os.Getenv("SLACK_BOT_TOKEN")}[0],
		Receiver: httpReceiver,
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
	fmt.Println("⚡️ Bolt app with HTTP receiver and custom properties is running!")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
