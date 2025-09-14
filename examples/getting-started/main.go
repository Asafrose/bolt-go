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
	"github.com/slack-go/slack"
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

	// Add a global middleware
	boltApp.Use(func(args types.AllMiddlewareArgs) error {
		// This middleware just calls next, similar to the JS example
		return args.Next()
	})

	// Listen to incoming messages that contain "hello"
	boltApp.Message("hello", func(args types.SlackEventMiddlewareArgs) error {
		// Check if we have a message event
		if args.Message != nil {
			// Filter out message events with subtypes (see https://api.slack.com/events/message)
			if args.Message.SubType == "" || args.Message.SubType == "bot_message" {
				// Create blocks for the message
				blocks := []slack.Block{
					&slack.SectionBlock{
						Type: slack.MBTSection,
						Text: &slack.TextBlockObject{
							Type: slack.MarkdownType,
							Text: fmt.Sprintf("Hey there <@%s>!", args.Message.User),
						},
						Accessory: &slack.Accessory{
							ButtonElement: &slack.ButtonBlockElement{
								Type: slack.METButton,
								Text: &slack.TextBlockObject{
									Type: slack.PlainTextType,
									Text: "Click Me",
								},
								ActionID: "button_click",
							},
						},
					},
				}

				// Use say function to respond
				text := fmt.Sprintf("Hey there <@%s>!", args.Message.User)
				_, err := args.Say(&types.SayArguments{
					Text:   text,
					Blocks: blocks,
				})
				return err
			}
		}
		return nil
	})

	// Listen to button clicks
	boltApp.Action(types.ActionConstraints{ActionID: "button_click"}, func(args types.SlackActionMiddlewareArgs) error {
		// Acknowledge the action
		if err := args.Ack(nil); err != nil {
			return err
		}

		// Extract user ID from context
		if args.Context != nil && args.Context.UserID != "" {
			// Respond to the button click if say function is available
			if args.Say != nil {
				text := fmt.Sprintf("<@%s> clicked the button", args.Context.UserID)
				_, err := args.Say(&types.SayArguments{
					Text: text,
				})
				return err
			}
		}
		return nil
	})

	// Start the app
	fmt.Println("⚡️ Bolt app is running!")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
