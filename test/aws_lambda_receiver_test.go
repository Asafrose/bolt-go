package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwsLambdaReceiver(t *testing.T) {
	t.Run("should create AWS Lambda receiver with valid options", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		assert.NotNil(t, receiver, "AWS Lambda receiver should be created")
	})

	t.Run("should initialize with app", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		assert.NoError(t, err, "AWS Lambda receiver should initialize with app")
	})

	t.Run("should handle process before response option", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret:         fakeSigningSecret,
			ProcessBeforeResponse: true,
		})

		assert.NotNil(t, receiver, "AWS Lambda receiver should be created with process before response")

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		assert.NoError(t, err, "Receiver should initialize with process before response")
	})

	t.Run("should handle custom properties", func(t *testing.T) {
		customProps := map[string]interface{}{
			"custom_key":  "custom_value",
			"timeout":     30,
			"memory_size": 512,
		}

		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret:    fakeSigningSecret,
			CustomProperties: customProps,
		})

		assert.NotNil(t, receiver, "AWS Lambda receiver should be created with custom properties")

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		assert.NoError(t, err, "Receiver should initialize with custom properties")
	})
}

func TestAwsLambdaReceiverEventHandling(t *testing.T) {
	t.Run("should handle API Gateway event", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create mock API Gateway event
		slackEvent := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"user":    "U123456",
				"text":    "<@U987654> hello",
				"channel": "C123456",
			},
		}

		eventBody, _ := json.Marshal(slackEvent)

		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/events",
			Headers: map[string]string{
				"Content-Type":              "application/json",
				"X-Slack-Signature":         "v0=test-signature",
				"X-Slack-Request-Timestamp": "1234567890",
			},
			Body: string(eventBody),
		}

		// Process the event
		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		// Should handle the event (may fail signature verification but shouldn't panic)
		assert.NoError(t, err, "Should handle Lambda event without panicking")
		assert.NotZero(t, response.StatusCode, "Should return a response")
	})

	t.Run("should handle different HTTP methods", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Test different HTTP methods
		methods := []string{"GET", "PUT", "DELETE"}

		for _, method := range methods {
			apiGatewayEvent := receivers.APIGatewayProxyEvent{
				HTTPMethod: method,
				Path:       "/slack/events",
				Headers:    map[string]string{},
				Body:       "",
			}

			ctx := context.Background()
			response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

			assert.NoError(t, err, "Should handle %s method", method)
			if method != "POST" {
				assert.Equal(t, 405, response.StatusCode, "Should return 405 for non-POST methods")
			}
		}
	})

	t.Run("should handle different paths", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Test different paths
		paths := []string{"/slack/events", "/slack/commands", "/slack/actions", "/slack/options"}

		for _, path := range paths {
			apiGatewayEvent := receivers.APIGatewayProxyEvent{
				HTTPMethod: "POST",
				Path:       path,
				Headers:    map[string]string{},
				Body:       "",
			}

			ctx := context.Background()
			response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

			assert.NoError(t, err, "Should handle path %s", path)
			assert.NotZero(t, response.StatusCode, "Should return a response for path %s", path)
		}
	})
}

func TestAwsLambdaReceiverResponses(t *testing.T) {
	t.Run("should format response correctly", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create a simple event
		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/events",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"type":"event_callback"}`,
		}

		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		assert.NoError(t, err, "Should handle event")
		assert.NotZero(t, response.StatusCode, "Should have status code")
		assert.NotNil(t, response.Headers, "Should have headers")
		assert.NotEmpty(t, response.Body, "Should have body")
	})

	t.Run("should handle URL verification", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create URL verification event
		verificationEvent := map[string]interface{}{
			"type":      "url_verification",
			"challenge": "test-challenge-string",
			"token":     "verification-token",
		}

		eventBody, _ := json.Marshal(verificationEvent)

		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/events",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(eventBody),
		}

		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		assert.NoError(t, err, "Should handle URL verification")
		assert.Equal(t, 200, response.StatusCode, "Should return 200 for URL verification")
		assert.Equal(t, "test-challenge-string", response.Body, "Should return challenge")
		assert.Equal(t, "text/plain", response.Headers["Content-Type"], "Should have correct content type")
	})

	t.Run("should handle error responses", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Don't initialize with app to trigger error

		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/events",
			Headers:    map[string]string{},
			Body:       "",
		}

		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		assert.NoError(t, err, "Should not return error from HandleLambdaEvent")
		assert.Equal(t, 500, response.StatusCode, "Should return 500 for uninitialized receiver")
		assert.Contains(t, response.Body, "error", "Error response should contain error message")
	})
}

func TestAwsLambdaReceiverConfiguration(t *testing.T) {
	t.Run("should handle different signing secrets", func(t *testing.T) {
		signingSecrets := []string{
			"short_secret",
			"very_long_signing_secret_with_many_characters_for_testing_purposes",
			"special!@#$%^&*()characters",
		}

		for _, secret := range signingSecrets {
			receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
				SigningSecret: secret,
			})

			assert.NotNil(t, receiver, "Receiver should be created with signing secret: %s", secret)
		}
	})

	t.Run("should handle missing signing secret", func(t *testing.T) {
		// Test with empty signing secret
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: "",
		})

		// Should still create receiver (validation might happen during processing)
		assert.NotNil(t, receiver, "Receiver should be created even with empty signing secret")
	})

	t.Run("should handle logger configuration", func(t *testing.T) {
		customLogger := "custom_logger"

		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
			Logger:        customLogger,
		})

		assert.NotNil(t, receiver, "Receiver should be created with custom logger")
	})
}

func TestAwsLambdaReceiverIntegration(t *testing.T) {
	t.Run("should integrate with bolt app", func(t *testing.T) {
		// Create app with AWS Lambda receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
			Receiver: receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
				SigningSecret: fakeSigningSecret,
			}),
		})
		require.NoError(t, err)

		// Add event handler
		handlerCalled := false
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			handlerCalled = true
			return nil
		})

		// Initialize app
		ctx := context.Background()
		err = app.Init(ctx)
		assert.NoError(t, err, "App should initialize with AWS Lambda receiver")

		// Verify handler registration
		assert.False(t, handlerCalled, "Handler should not be called yet")
	})

	t.Run("should handle app initialization errors", func(t *testing.T) {
		// Create app with invalid configuration
		_, err := bolt.New(bolt.AppOptions{
			// Missing required fields
			Receiver: receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
				SigningSecret: fakeSigningSecret,
			}),
		})

		// Should return error for invalid configuration
		assert.Error(t, err, "Should return error for invalid app configuration")
	})

	t.Run("should handle form-encoded slash commands", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create form-encoded slash command
		formData := "token=verification-token&team_id=T123456&team_domain=testteam&channel_id=C123456&channel_name=general&user_id=U123456&user_name=testuser&command=/test&text=hello+world&response_url=https://hooks.slack.com/commands/T123456/123456/abcdef&trigger_id=123456.123456.abcdef"

		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/commands",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: formData,
		}

		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		assert.NoError(t, err, "Should handle form-encoded command")
		assert.NotZero(t, response.StatusCode, "Should return a response")
	})

	t.Run("should handle interactive component payloads", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		// Create interactive component payload
		actionPayload := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"type":      "button",
				},
			},
			"user":    map[string]interface{}{"id": "U123456"},
			"channel": map[string]interface{}{"id": "C123456"},
		}

		payloadBytes, _ := json.Marshal(actionPayload)
		formData := "payload=" + string(payloadBytes)

		apiGatewayEvent := receivers.APIGatewayProxyEvent{
			HTTPMethod: "POST",
			Path:       "/slack/actions",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: formData,
		}

		ctx := context.Background()
		response, err := receiver.HandleLambdaEvent(ctx, apiGatewayEvent)

		assert.NoError(t, err, "Should handle interactive component payload")
		assert.NotZero(t, response.StatusCode, "Should return a response")
	})
}
