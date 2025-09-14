package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/samber/lo"
)

func main() {
	// Get required environment variables
	token := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	if token == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable is required")
	}
	if appToken == "" {
		log.Fatal("SLACK_APP_TOKEN environment variable is required")
	}

	// Initialize the app with Socket Mode
	boltApp, err := app.New(app.AppOptions{
		Token:      lo.ToPtr(token),
		AppToken:   lo.ToPtr(appToken),
		SocketMode: true,
		LogLevel:   lo.ToPtr(bolt.LogLevelDebug),
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Example 1: Using typed event constants (RECOMMENDED)
	// This provides type safety and prevents typos
	boltApp.Event(types.EventTypeAppMention, func(args types.SlackEventMiddlewareArgs) error {
		fmt.Println("✅ Received app_mention event using typed constant")
		if args.Say != nil {
			_, err := args.Say(&types.SayArguments{
				Text: "Hello! You mentioned me using a typed event constant.",
			})
			return err
		}
		return nil
	})

	// Example 2: Using custom event types
	// For new or custom events not yet in the predefined constants
	customEventType := types.SlackEventType("my_custom_event")
	boltApp.Event(customEventType, func(args types.SlackEventMiddlewareArgs) error {
		fmt.Println("🔧 Received custom event type")
		args.Logger.Info("Custom event received", "type", customEventType.String())
		return nil
	})

	// Example 3: Using message event with typed constant
	boltApp.Event(types.EventTypeMessage, func(args types.SlackEventMiddlewareArgs) error {
		fmt.Println("📝 Received message event using typed constant")
		// Only respond to direct messages to avoid spam
		if args.Message != nil && args.Message.ChannelType == "im" {
			if args.Say != nil {
				_, err := args.Say(&types.SayArguments{
					Text: "Hello! You sent me a direct message using a typed constant.",
				})
				return err
			}
		}
		return nil
	})

	// Example 4: Multiple event types with constants
	eventHandlers := map[types.SlackEventType]string{
		types.EventTypeReactionAdded:            "Someone added a reaction! 👍",
		types.EventTypeReactionRemoved:          "Someone removed a reaction! 👎",
		types.EventTypeChannelCreated:           "A new channel was created! 🎉",
		types.EventTypeChannelArchive:           "A channel was archived! 📦",
		types.EventTypeTeamJoin:                 "Someone joined the team! 🎊",
		types.EventTypeMessageMetadataPosted:    "Message metadata was posted! 📝",
		types.EventTypeMessageMetadataUpdated:   "Message metadata was updated! ✏️",
		types.EventTypeMessageMetadataDeleted:   "Message metadata was deleted! 🗑️",
	}

	for eventType, message := range eventHandlers {
		// Capture the message in the closure
		msg := message
		boltApp.Event(eventType, func(args types.SlackEventMiddlewareArgs) error {
			fmt.Printf("📢 Event: %s - %s\n", eventType.String(), msg)
			args.Logger.Info("Event received", "type", eventType.String(), "message", msg)
			return nil
		})
	}

	// Example 5: Demonstrating event type validation
	fmt.Println("\n🔍 Event Type Validation Examples:")
	
	// Valid event types
	validEvents := []types.SlackEventType{
		types.EventTypeMessage,
		types.EventTypeAppMention,
		types.EventTypeFunctionExecuted,
		types.EventTypeWorkflowStepExecute,
	}
	
	for _, eventType := range validEvents {
		fmt.Printf("✅ %s is valid: %t\n", eventType.String(), eventType.IsValid())
	}
	
	// Invalid event types
	invalidEvents := []types.SlackEventType{
		types.SlackEventType("invalid_event"),
		types.SlackEventType("not_real"),
		types.SlackEventType("typo_in_event_name"),
	}
	
	for _, eventType := range invalidEvents {
		fmt.Printf("❌ %s is valid: %t\n", eventType.String(), eventType.IsValid())
	}

	// Example 6: List all available event types
	fmt.Printf("\n📋 Total available event types: %d\n", len(types.AllEventTypes()))
	fmt.Println("First 10 event types:")
	for i, eventType := range types.AllEventTypes()[:10] {
		fmt.Printf("  %d. %s\n", i+1, eventType.String())
	}

	// Start your app
	fmt.Println("\n⚡️ Bolt app started with event type demonstrations!")
	fmt.Println("Try mentioning the bot or sending a direct message to see the different approaches in action.")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
