package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	bolt "github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/oauth"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// GinReceiver implements a custom receiver using the Gin web framework
type GinReceiver struct {
	signingSecret     string
	installationStore oauth.InstallationStore
	clientID          string
	clientSecret      string
	scopes            []string
	router            *gin.Engine
	boltApp           *app.App
}

// NewGinReceiver creates a new Gin-based receiver
func NewGinReceiver(config *GinReceiverConfig) (*GinReceiver, error) {
	if config.Router == nil {
		config.Router = gin.Default()
	}

	receiver := &GinReceiver{
		signingSecret:     config.SigningSecret,
		installationStore: config.InstallationStore,
		clientID:          config.ClientID,
		clientSecret:      config.ClientSecret,
		scopes:            config.Scopes,
		router:            config.Router,
	}

	// Set up OAuth routes
	receiver.setupRoutes()

	return receiver, nil
}

type GinReceiverConfig struct {
	SigningSecret     string
	ClientID          string
	ClientSecret      string
	Scopes            []string
	InstallationStore oauth.InstallationStore
	Router            *gin.Engine
}

func (r *GinReceiver) setupRoutes() {
	// Redirect root to install
	r.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/slack/install")
	})

	// OAuth install endpoint
	r.router.GET("/slack/install", func(c *gin.Context) {
		c.HTML(http.StatusOK, "install.html", gin.H{
			"title": "Install Slack App",
		})
	})

	// OAuth callback endpoint
	r.router.GET("/slack/oauth_redirect", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OAuth callback received",
		})
	})

	// Slack events endpoint
	r.router.POST("/slack/events", func(c *gin.Context) {
		// In a real implementation, this would process Slack events
		// For now, we'll just acknowledge
		c.JSON(http.StatusOK, gin.H{
			"challenge": c.Query("challenge"),
		})
	})
}

// Init initializes the receiver with the Bolt app
func (r *GinReceiver) Init(app types.App) error {
	// Store reference to app for processing events
	return nil
}

// Start starts the receiver
func (r *GinReceiver) Start(ctx context.Context) error {
	return r.router.Run(":3000")
}

// Stop stops the receiver
func (r *GinReceiver) Stop(ctx context.Context) error {
	// Gin doesn't have a built-in stop method
	return nil
}

func main() {
	if os.Getenv("SLACK_SIGNING_SECRET") == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not found!")
	}

	// Create Gin router
	router := gin.Default()

	// Create custom Gin receiver
	receiver, err := NewGinReceiver(&GinReceiverConfig{
		SigningSecret:     os.Getenv("SLACK_SIGNING_SECRET"),
		ClientID:          os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret:      os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes:            []string{"commands", "chat:write", "app_mentions:read"},
		InstallationStore: oauth.NewMemoryInstallationStore(),
		Router:            router,
	})
	if err != nil {
		log.Fatalf("Failed to create Gin receiver: %v", err)
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
	// Note: Once you update to the latest version, you can use:
	// boltApp.Event(types.SlackEventType("app_mention"), ...)
	boltApp.Event(types.EventTypeAppMention, func(args types.SlackEventMiddlewareArgs) error {

		userIdStr := args.Context.UserID

		text := fmt.Sprintf("<@%s> Hi there :wave:", userIdStr)
		_, err := args.Say(&types.SayArguments{
			Text: text,
		})

		if err != nil {
			return err
		}

		return nil
	})

	// Start the app
	fmt.Println("⚡️ Bolt app with Gin receiver is running!")
	if err := boltApp.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
