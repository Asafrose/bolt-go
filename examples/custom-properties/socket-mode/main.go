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
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		log.Fatal("SLACK_APP_TOKEN environment variable is required")
	}

	// Create Socket Mode receiver with custom properties extractor
	socketReceiver := receivers.NewSocketModeReceiver(types.SocketModeReceiverOptions{
		AppToken: appToken,
		CustomPropertiesExtractor: func(payload map[string]interface{}) map[string]interface{} {
			return map[string]interface{}{
				"socket_mode_payload_type": payload["type"],
				"socket_mode_payload":      payload,
				"foo":                      "bar",
			}
		},
		CustomProperties: map[string]interface{}{
			"example_property": "custom_socket_value",
		},
	})

	// Initialize the app with custom receiver
	boltApp, err := app.New(app.AppOptions{
		Token:    &[]string{os.Getenv("SLACK_BOT_TOKEN")}[0],
		Receiver: socketReceiver,
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Add middleware that logs the context (including custom properties)
	boltApp.Use(func(args types.AllMiddlewareArgs) error {
		args.Logger.Info("Request context with custom properties", "context", args.Context)
		return args.Next()
	})

	// Start your app
	fmt.Println("⚡️ Bolt app with Socket Mode receiver and custom properties is running!")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
