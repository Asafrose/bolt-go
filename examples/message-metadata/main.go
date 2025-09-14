package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
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
		Token:      token,
		AppToken:   appToken,
		SocketMode: true,
		LogLevel:   bolt.LogLevelDebug,
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Listen to slash command
	// Post a message with Message Metadata
	boltApp.Command("/post", func(args types.SlackCommandMiddlewareArgs) error {
		if err := args.Ack(nil); err != nil {
			return err
		}

		// Create message with metadata
		text := "Message Metadata Posting"
		_, err := args.Say(&types.SayArguments{
			Text: text,
			// Note: Message metadata would need to be added to SayArguments
			// This is a simplified version - the actual implementation would need
			// to support metadata in the SayArguments type
		})
		return err
	})

	// Listen for message_metadata_posted event
	boltApp.Event(types.EventTypeMessageMetadataPosted, func(args types.SlackEventMiddlewareArgs) error {
		// Extract event data from generic event
		if genericEvent, ok := args.Event.(*helpers.GenericSlackEvent); ok {
			if messageTS, exists := genericEvent.RawData["message_ts"]; exists {
				if metadata, exists := genericEvent.RawData["metadata"]; exists {
					// Convert metadata to JSON string for display
					metadataJSON, err := json.Marshal(metadata)
					if err != nil {
						return fmt.Errorf("failed to marshal metadata: %w", err)
					}

					// Create response blocks
					blocks := []slack.Block{
						&slack.SectionBlock{
							Type: slack.MBTSection,
							Text: &slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "Message Metadata Posted",
							},
						},
						&slack.ContextBlock{
							Type: slack.MBTContext,
							ContextElements: slack.ContextElements{
								Elements: []slack.MixedElement{
									&slack.TextBlockObject{
										Type: slack.MarkdownType,
										Text: string(metadataJSON),
									},
								},
							},
						},
					}

					// Reply in thread
					text := "Message Metadata Posted"
					threadTS := ""
					if ts, ok := messageTS.(string); ok {
						threadTS = ts
					}

					_, err = args.Say(&types.SayArguments{
						Text:     text,
						Blocks:   blocks,
						ThreadTS: threadTS,
					})
					return err
				}
			}
		}
		return nil
	})

	// Start your app
	fmt.Println("⚡️ Bolt app started")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
