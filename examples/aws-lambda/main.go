package main

import (
	"fmt"
	"os"

	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/samber/lo"
	"github.com/slack-go/slack"
)

var boltApp *app.App
var awsLambdaReceiver *receivers.AwsLambdaReceiver

func init() {
	// Get required environment variables
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	token := os.Getenv("SLACK_BOT_TOKEN")

	if signingSecret == "" {
		panic("SLACK_SIGNING_SECRET environment variable is required")
	}
	if token == "" {
		panic("SLACK_BOT_TOKEN environment variable is required")
	}

	// Initialize your custom receiver
	awsLambdaReceiver = receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
		SigningSecret: signingSecret,
	})

	// Initializes your app with your bot token and the AWS Lambda ready receiver
	var err error
	boltApp, err = app.New(app.AppOptions{
		Token:    lo.ToPtr(token),
		Receiver: awsLambdaReceiver,

		// When using the AwsLambdaReceiver, processBeforeResponse can be omitted.
		// If you use other Receivers, such as HTTPReceiver for OAuth flow support
		// then processBeforeResponse: true is required. This option will defer sending back
		// the acknowledgement until after your handler has run to ensure your function
		// isn't terminated early by responding to the HTTP request that triggered it.

		// ProcessBeforeResponse: true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create app: %v", err))
	}

	// Listens to incoming messages that contain "hello"
	boltApp.Message("hello", func(args types.SlackEventMiddlewareArgs) error {
		if args.Message != nil {
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
				Text:   lo.ToPtr(text),
				Blocks: blocks,
			})
			return err
		}
		return nil
	})

	// Listens for an action from a button click
	boltApp.Action(types.ActionConstraints{ActionID: lo.ToPtr("button_click")}, func(args types.SlackActionMiddlewareArgs) error {
		if err := args.Ack(nil); err != nil {
			return err
		}

		// Extract user ID from context
		if args.Context != nil && args.Context.UserID != nil {
			// Respond to the button click if say function is available
			if args.Say != nil {
				text := fmt.Sprintf("<@%s> clicked the button", *args.Context.UserID)
				_, err := (*args.Say)(&types.SayArguments{
					Text: lo.ToPtr(text),
				})
				return err
			}
		}
		return nil
	})

	// Listens to incoming messages that contain "goodbye"
	boltApp.Message("goodbye", func(args types.SlackEventMiddlewareArgs) error {
		if args.Message != nil {
			text := fmt.Sprintf("See ya later, <@%s> :wave:", args.Message.User)
			_, err := args.Say(&types.SayArguments{
				Text: lo.ToPtr(text),
			})
			return err
		}
		return nil
	})
}

func main() {
	// Convert the receiver to an AWS Lambda handler
	handler := awsLambdaReceiver.ToHandler()

	// Start the Lambda handler
	lambda.Start(handler)
}
