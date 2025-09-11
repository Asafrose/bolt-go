package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/samber/lo"
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

	// Publish an App Home
	boltApp.Event("app_home_opened", func(args types.SlackEventMiddlewareArgs) error {
		// Extract event data
		if eventMap, ok := args.Event.(map[string]interface{}); ok {
			if userID, exists := eventMap["user"]; exists {
				if userIDStr, ok := userID.(string); ok {
					// Create home view blocks
					blocks := []slack.Block{
						&slack.SectionBlock{
							Type:    slack.MBTSection,
							BlockID: "section678",
							Text: &slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "App Home Published",
							},
						},
					}

					// Use the client to publish view
					_, err := args.Client.PublishView(userIDStr, slack.HomeTabViewRequest{
						Type:   slack.VTHomeTab,
						Blocks: slack.Blocks{BlockSet: blocks},
					}, "")
					return err
				}
			}
		}
		return nil
	})

	// Message Shortcut example
	boltApp.Shortcut(types.ShortcutConstraints{CallbackID: lo.ToPtr("launch_msg_shortcut")}, func(args types.SlackShortcutMiddlewareArgs) error {
		if err := args.Ack(nil); err != nil {
			return err
		}

		args.Logger.Info("Message shortcut triggered", "shortcut", args.Shortcut)
		return nil
	})

	// Global Shortcut example
	// setup global shortcut in App config with `launch_shortcut` as callback id
	// add `commands` scope
	boltApp.Shortcut(types.ShortcutConstraints{CallbackID: lo.ToPtr("launch_shortcut")}, func(args types.SlackShortcutMiddlewareArgs) error {
		// Acknowledge shortcut request
		if err := args.Ack(nil); err != nil {
			return err
		}

		// Extract trigger_id from shortcut
		if shortcutMap, ok := args.Shortcut.(map[string]interface{}); ok {
			if triggerID, exists := shortcutMap["trigger_id"]; exists {
				if triggerIDStr, ok := triggerID.(string); ok {
					// Create modal blocks
					blocks := []slack.Block{
						&slack.SectionBlock{
							Type: slack.MBTSection,
							Text: &slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "About the simplest modal you could conceive of :smile:\n\nMaybe <https://api.slack.com/reference/block-kit/interactive-components|*make the modal interactive*> or <https://api.slack.com/surfaces/modals/using#modifying|*learn more advanced modal use cases*>.",
							},
						},
						&slack.ContextBlock{
							Type: slack.MBTContext,
							ContextElements: slack.ContextElements{
								Elements: []slack.MixedElement{
									&slack.TextBlockObject{
										Type: slack.MarkdownType,
										Text: "Psssst this modal was designed using <https://api.slack.com/tools/block-kit-builder|*Block Kit Builder*>",
									},
								},
							},
						},
					}

					// Open modal
					_, err := args.Client.OpenView(triggerIDStr, slack.ModalViewRequest{
						Type: slack.VTModal,
						Title: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "My App",
						},
						Close: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Close",
						},
						Blocks: slack.Blocks{BlockSet: blocks},
					})
					if err != nil {
						args.Logger.Error("Failed to open modal", "error", err)
					}
					return err
				}
			}
		}
		return nil
	})

	// Subscribe to 'app_mention' event in your App config
	// need app_mentions:read and chat:write scopes
	boltApp.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
		if eventMap, ok := args.Event.(map[string]interface{}); ok {
			if userID, exists := eventMap["user"]; exists {
				if userIDStr, ok := userID.(string); ok {
					blocks := []slack.Block{
						&slack.SectionBlock{
							Type: slack.MBTSection,
							Text: &slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: fmt.Sprintf("Thanks for the mention <@%s>! Click my fancy button", userIDStr),
							},
							Accessory: &slack.Accessory{
								ButtonElement: &slack.ButtonBlockElement{
									Type: slack.METButton,
									Text: &slack.TextBlockObject{
										Type:  slack.PlainTextType,
										Text:  "Button",
										Emoji: lo.ToPtr(true),
									},
									Value:    "click_me_123",
									ActionID: "first_button",
								},
							},
						},
					}

					text := fmt.Sprintf("Thanks for the mention <@%s>! Click my fancy button", userIDStr)
					_, err := args.Say(&types.SayArguments{
						Text:   lo.ToPtr(text),
						Blocks: blocks,
					})
					if err != nil {
						args.Logger.Error("Failed to respond to app mention", "error", err)
					}
					return err
				}
			}
		}
		return nil
	})

	// Subscribe to `message.channels` event in your App Config
	// need channels:read scope
	boltApp.Message("hello", func(args types.SlackEventMiddlewareArgs) error {
		if args.Message != nil {
			blocks := []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf("Thanks for the mention <@%s>! Click my fancy button", args.Message.User),
					},
					Accessory: &slack.Accessory{
						ButtonElement: &slack.ButtonBlockElement{
							Type: slack.METButton,
							Text: &slack.TextBlockObject{
								Type:  slack.PlainTextType,
								Text:  "Button",
								Emoji: lo.ToPtr(true),
							},
							Value:    "click_me_123",
							ActionID: "first_button",
						},
					},
				},
			}

			text := fmt.Sprintf("Thanks for the mention <@%s>! Click my fancy button", args.Message.User)
			_, err := args.Say(&types.SayArguments{
				Text:   lo.ToPtr(text),
				Blocks: blocks,
			})
			return err
		}
		return nil
	})

	// Listen and respond to button click
	boltApp.Action(types.ActionConstraints{ActionID: lo.ToPtr("first_button")}, func(args types.SlackActionMiddlewareArgs) error {
		args.Logger.Info("button clicked", "action", args.Action)

		// acknowledge the request right away
		if err := args.Ack(nil); err != nil {
			return err
		}

		// Respond to the button click if say function is available
		if args.Say != nil {
			text := "Thanks for clicking the fancy button"
			_, err := (*args.Say)(&types.SayArguments{
				Text: lo.ToPtr(text),
			})
			return err
		}
		return nil
	})

	// Listen to slash command
	// need to add commands permission
	// create slash command in App Config
	boltApp.Command("/socketslash", func(args types.SlackCommandMiddlewareArgs) error {
		// Acknowledge command request
		if err := args.Ack(nil); err != nil {
			return err
		}

		text := args.Command.Text
		_, err := args.Say(&types.SayArguments{
			Text: lo.ToPtr(text),
		})
		return err
	})

	// Start your app
	fmt.Println("⚡️ Bolt app is running!")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
