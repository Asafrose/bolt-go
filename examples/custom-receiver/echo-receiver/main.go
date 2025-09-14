package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/samber/lo"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// EchoReceiver implements a custom receiver using the Echo web framework
type EchoReceiver struct {
	signingSecret     string
	installationStore oauth.InstallationStore
	clientID          string
	clientSecret      string
	scopes            []string
	echo              *echo.Echo
	boltApp           *app.App
}

// NewEchoReceiver creates a new Echo-based receiver
func NewEchoReceiver(config *EchoReceiverConfig) (*EchoReceiver, error) {
	if config.Echo == nil {
		config.Echo = echo.New()
		config.Echo.Use(middleware.Logger())
		config.Echo.Use(middleware.Recover())
	}

	receiver := &EchoReceiver{
		signingSecret:     config.SigningSecret,
		installationStore: config.InstallationStore,
		clientID:          config.ClientID,
		clientSecret:      config.ClientSecret,
		scopes:            config.Scopes,
		echo:              config.Echo,
	}

	// Set up OAuth routes
	receiver.setupRoutes()

	return receiver, nil
}

type EchoReceiverConfig struct {
	SigningSecret     string
	ClientID          string
	ClientSecret      string
	Scopes            []string
	InstallationStore oauth.InstallationStore
	Echo              *echo.Echo
}

func (r *EchoReceiver) setupRoutes() {
	// Redirect root to install
	r.echo.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/slack/install")
	})

	// OAuth install endpoint
	r.echo.GET("/slack/install", func(c echo.Context) error {
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Install Slack App</title>
		</head>
		<body>
			<h1>Install Slack App</h1>
			<p>Click the button below to install the app to your Slack workspace.</p>
			<a href="/slack/oauth_redirect?code=dummy">
				<img alt="Add to Slack" height="40" width="139" 
				     src="https://platform.slack-edge.com/img/add_to_slack.png" 
				     srcSet="https://platform.slack-edge.com/img/add_to_slack.png 1x, https://platform.slack-edge.com/img/add_to_slack@2x.png 2x">
			</a>
		</body>
		</html>
		`
		return c.HTML(http.StatusOK, html)
	})

	// OAuth callback endpoint
	r.echo.GET("/slack/oauth_redirect", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "OAuth callback received",
			"code":    c.QueryParam("code"),
		})
	})

	// Slack events endpoint
	r.echo.POST("/slack/events", func(c echo.Context) error {
		// Handle URL verification challenge
		if challenge := c.QueryParam("challenge"); challenge != "" {
			return c.JSON(http.StatusOK, map[string]string{
				"challenge": challenge,
			})
		}

		// In a real implementation, this would process Slack events
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Custom route example
	r.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"app":    "bolt-go-echo-receiver",
		})
	})
}

// Init initializes the receiver with the Bolt app
func (r *EchoReceiver) Init(app types.App) error {
	// Store reference to app for processing events
	return nil
}

// Start starts the receiver
func (r *EchoReceiver) Start(ctx context.Context) error {
	return r.echo.Start(":3000")
}

// Stop stops the receiver
func (r *EchoReceiver) Stop(ctx context.Context) error {
	return r.echo.Shutdown(ctx)
}

func main() {
	if os.Getenv("SLACK_SIGNING_SECRET") == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not found!")
	}

	// Create Echo instance
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Create custom Echo receiver
	receiver, err := NewEchoReceiver(&EchoReceiverConfig{
		SigningSecret:     os.Getenv("SLACK_SIGNING_SECRET"),
		ClientID:          os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret:      os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes:            []string{"commands", "chat:write", "app_mentions:read"},
		InstallationStore: oauth.NewMemoryInstallationStore(),
		Echo:              e,
	})
	if err != nil {
		log.Fatalf("Failed to create Echo receiver: %v", err)
	}

	// Create Bolt app with custom receiver
	boltApp, err := app.New(app.AppOptions{
		LogLevel: lo.ToPtr(bolt.LogLevelDebug),
		Receiver: receiver,
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Set up Slack event handlers
	boltApp.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
		if eventMap, ok := args.Event.(map[string]interface{}); ok {
			if userID, exists := eventMap["user"]; exists {
				if userIDStr, ok := userID.(string); ok {
					text := fmt.Sprintf("<@%s> Hi there :wave:", userIDStr)
					_, err := args.Say(&types.SayArguments{
						Text: lo.ToPtr(text),
					})
					return err
				}
			}
		}
		return nil
	})

	// Start the app
	port := 3000
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			port = p
		}
	}

	fmt.Printf("⚡️ Bolt app with Echo receiver is running on port %d!\n", port)
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
