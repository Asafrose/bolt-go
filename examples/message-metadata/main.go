package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/samber/lo"
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
		Token:      lo.ToPtr(token),
		AppToken:   lo.ToPtr(appToken),
		SocketMode: true,
		LogLevel:   lo.ToPtr(app.LogLevelDebug),
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
			Text: lo.ToPtr(text),
			// Note: Message metadata would need to be added to SayArguments
			// This is a simplified version - the actual implementation would need
			// to support metadata in the SayArguments type
		})
		return err
	})

	// Listen for message_metadata_posted event
	boltApp.Event("message_metadata_posted", func(args types.SlackEventMiddlewareArgs) error {
		// Extract event data
		if eventMap, ok := args.Event.(map[string]interface{}); ok {
			if messageTS, exists := eventMap["message_ts"]; exists {
				if metadata, exists := eventMap["metadata"]; exists {
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
						Text:     lo.ToPtr(text),
						Blocks:   blocks,
						ThreadTS: lo.ToPtr(threadTS),
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
