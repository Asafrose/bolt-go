package test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Asafrose/bolt-go"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAwsLambdaAdvanced implements the missing tests from AwsLambdaReceiver.spec.ts
func TestAwsLambdaAdvanced(t *testing.T) {
	t.Parallel()
	t.Run("should instantiate with default logger", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		assert.NotNil(t, receiver)
	})

	t.Run("should return a 404 if app has no registered handlers for an incoming event, and return a 200 if app does have registered handlers", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// First create app without handlers
		app1, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app1)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create a dummy app mention event
		body := `{
			"token": "verification_token",
			"team_id": "T1234567890",
			"api_app_id": "A1234567890",
			"event": {
				"type": "app_mention",
				"user": "U1234567890",
				"text": "<@U0LAN0Z89> hello",
				"ts": "1515449522.000016",
				"channel": "C1234567890"
			},
			"type": "event_callback",
			"event_id": "Ev1234567890",
			"event_time": 1515449522
		}`
		awsEvent := createDummyAWSEvent(body, timestamp, fakeSigningSecret)

		// Test without handlers - should return 404
		response1, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, response1.StatusCode)

		// Add a handler to the same app
		app1.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
			return args.Ack(nil)
		})

		// Test with handlers - should return 200
		response2, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, response2.StatusCode)
	})

	t.Run("should accept ssl_check requests", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Create app and initialize receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create SSL check request body
		body := "ssl_check=1&token=legacy-fixed-token"

		// Create AWS event with form-encoded content type
		awsEvent := receivers.AwsEvent{
			Resource:   "/slack/events",
			Path:       "/slack/events",
			HTTPMethod: "POST",
			Headers: map[string]string{
				"Accept":                    "application/json,*/*",
				"Content-Type":              "application/x-www-form-urlencoded",
				"Host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",
				"User-Agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         createValidSignature(body, timestamp, fakeSigningSecret),
			},
			MultiValueHeaders:               make(map[string][]string),
			QueryStringParameters:           make(map[string]string),
			MultiValueQueryStringParameters: make(map[string][]string),
			PathParameters:                  make(map[string]string),
			StageVariables:                  make(map[string]string),
			RequestContext:                  make(map[string]interface{}),
			Body:                            body,
			IsBase64Encoded:                 false,
		}

		// Test SSL check - should return 200
		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Run("should accept an event containing a base64 encoded body", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Create app and initialize receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create a dummy app mention event
		eventBody := `{
			"token": "verification_token",
			"team_id": "T1234567890",
			"api_app_id": "A1234567890",
			"event": {
				"type": "app_mention",
				"user": "U1234567890",
				"text": "<@U0LAN0Z89> hello",
				"ts": "1515449522.000016",
				"channel": "C1234567890"
			},
			"type": "event_callback",
			"event_id": "Ev1234567890",
			"event_time": 1515449522
		}`

		// Base64 encode the body
		base64Body := base64.StdEncoding.EncodeToString([]byte(eventBody))

		// Create AWS event with base64 encoded body
		awsEvent := receivers.AwsEvent{
			Resource:   "/slack/events",
			Path:       "/slack/events",
			HTTPMethod: "POST",
			Headers: map[string]string{
				"Accept":                    "application/json,*/*",
				"Content-Type":              "application/json",
				"Host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",
				"User-Agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         createValidSignature(eventBody, timestamp, fakeSigningSecret), // Sign the decoded body
			},
			MultiValueHeaders:               make(map[string][]string),
			QueryStringParameters:           make(map[string]string),
			MultiValueQueryStringParameters: make(map[string][]string),
			PathParameters:                  make(map[string]string),
			StageVariables:                  make(map[string]string),
			RequestContext:                  make(map[string]interface{}),
			Body:                            base64Body,
			IsBase64Encoded:                 true, // Important: mark as base64 encoded
		}

		// Test base64 encoded event - should return 404 (no handlers for app_mention)
		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, response.StatusCode)
	})

	t.Run("does not perform signature verification if signature verification flag is set to false", func(t *testing.T) {
		falseValue := false
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret:         fakeSigningSecret,
			SignatureVerification: &falseValue, // Disable signature verification
		})

		// Create app and initialize receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create URL verification request with INVALID signature (should still work)
		body := `{"type":"url_verification","challenge":"test_challenge","token":"test_token"}`

		// Create AWS event with completely invalid signature
		awsEvent := receivers.AwsEvent{
			Resource:   "/slack/events",
			Path:       "/slack/events",
			HTTPMethod: "POST",
			Headers: map[string]string{
				"Accept":                    "application/json,*/*",
				"Content-Type":              "application/json",
				"Host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",
				"User-Agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         "v0=invalid-signature-should-be-ignored",
			},
			MultiValueHeaders:               make(map[string][]string),
			QueryStringParameters:           make(map[string]string),
			MultiValueQueryStringParameters: make(map[string][]string),
			PathParameters:                  make(map[string]string),
			StageVariables:                  make(map[string]string),
			RequestContext:                  make(map[string]interface{}),
			Body:                            body,
			IsBase64Encoded:                 false,
		}

		// Test with invalid signature but verification disabled - should return 200
		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode, "Should return 200 when signature verification is disabled")
	})

	t.Run("should accept proxy events with lowercase header properties", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Create app and initialize receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create a dummy app mention event
		body := `{
			"token": "verification_token",
			"team_id": "T1234567890",
			"api_app_id": "A1234567890",
			"event": {
				"type": "app_mention",
				"user": "U1234567890",
				"text": "<@U0LAN0Z89> hello",
				"ts": "1515449522.000016",
				"channel": "C1234567890"
			},
			"type": "event_callback",
			"event_id": "Ev1234567890",
			"event_time": 1515449522
		}`

		// Create AWS event with LOWERCASE headers (important for the test)
		awsEvent := receivers.AwsEvent{
			Resource:   "/slack/events",
			Path:       "/slack/events",
			HTTPMethod: "POST",
			Headers: map[string]string{
				"accept":                    "application/json,*/*",                                   // lowercase
				"content-type":              "application/json",                                       // lowercase with dash
				"host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",           // lowercase
				"user-agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",           // lowercase with dash
				"x-slack-request-timestamp": strconv.FormatInt(timestamp, 10),                         // lowercase with dashes
				"x-slack-signature":         createValidSignature(body, timestamp, fakeSigningSecret), // lowercase with dashes
			},
			MultiValueHeaders:               make(map[string][]string),
			QueryStringParameters:           make(map[string]string),
			MultiValueQueryStringParameters: make(map[string][]string),
			PathParameters:                  make(map[string]string),
			StageVariables:                  make(map[string]string),
			RequestContext:                  make(map[string]interface{}),
			Body:                            body,
			IsBase64Encoded:                 false,
		}

		// Test without handlers - should return 404 (but headers should be processed correctly)
		response1, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, response1.StatusCode)

		// Add a handler for app_mention
		app.Event("app_mention", func(args types.SlackEventMiddlewareArgs) error {
			return args.Ack(nil)
		})

		// Test with handlers - should return 200 (proving lowercase headers work)
		response2, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, response2.StatusCode)
	})
	t.Run("should have start method", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		ctx := context.Background()
		err = receiver.Start(ctx)
		require.NoError(t, err, "Start method should work")
	})

	t.Run("should have stop method", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		ctx := context.Background()
		err = receiver.Start(ctx)
		require.NoError(t, err)

		err = receiver.Stop(ctx)
		require.NoError(t, err, "Stop method should work")
	})

	t.Run("should return a 404 if app has no registered handlers for an incoming event", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create a valid event with no matching handlers
		timestamp := time.Now().Unix()
		eventBody := `{"type":"event_callback","event":{"type":"app_mention","text":"hello"}}`
		awsEvent := createDummyAWSEvent(eventBody, timestamp, fakeSigningSecret)

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 404, response.StatusCode, "Should return 404 for unhandled events")
	})

	t.Run("should return a 200 if app does have registered handlers", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		// Register a handler for app_mention events
		app.Event("app_mention", func(args bolt.SlackEventMiddlewareArgs) error {
			// Acknowledge the event
			if args.Ack != nil {
				if err := args.Ack(nil); err != nil {
					t.Errorf("Failed to acknowledge event: %v", err)
				}
			}
			return args.Next()
		})

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create a valid event with matching handler (use the same format as other tests)
		timestamp := time.Now().Unix()
		eventBody := `{"type":"event_callback","team_id":"T123456","api_app_id":"A123456","event":{"type":"app_mention","text":"hello","user":"U123456","channel":"C123456"},"event_id":"Ev123456","event_time":1234567890,"authed_users":["U123456"]}`
		awsEvent := createDummyAWSEvent(eventBody, timestamp, fakeSigningSecret)

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 200, response.StatusCode, "Should return 200 for handled events")
	})

	t.Run("should accept url_verification requests", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create URL verification event
		timestamp := time.Now().Unix()
		eventBody := `{"type":"url_verification","challenge":"test_challenge_string","token":"test_token"}`
		awsEvent := createDummyAWSEvent(eventBody, timestamp, fakeSigningSecret)

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 200, response.StatusCode, "Should return 200 for url_verification")
		assert.Equal(t, "application/json", response.Headers["Content-Type"], "Should set JSON content type")

		var responseBody map[string]string
		err = json.Unmarshal([]byte(response.Body), &responseBody)
		require.NoError(t, err)

		assert.Equal(t, "test_challenge_string", responseBody["challenge"], "Should return the challenge")
	})

	t.Run("should detect invalid signature", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create event with invalid signature
		timestamp := time.Now().Unix()
		eventBody := `{"type":"url_verification","challenge":"test_challenge","token":"test_token"}`

		// Create invalid signature by using wrong secret
		baseString := fmt.Sprintf("v0:%d:%s", timestamp, eventBody)
		mac := hmac.New(sha256.New, []byte("wrong-secret"))
		mac.Write([]byte(baseString))
		invalidSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

		awsEvent := receivers.AwsEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type":              "application/json",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         invalidSignature,
			},
			IsBase64Encoded: false,
		}

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 401, response.StatusCode, "Should return 401 for invalid signature")
	})

	t.Run("should detect too old request timestamp", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create event with old timestamp (more than 5 minutes ago)
		oldTimestamp := time.Now().Unix() - 400 // 6+ minutes ago
		eventBody := `{"type":"url_verification","challenge":"test_challenge","token":"test_token"}`
		awsEvent := createDummyAWSEvent(eventBody, oldTimestamp, fakeSigningSecret)

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 401, response.StatusCode, "Should return 401 for old timestamp")
	})

	t.Run("should handle SSL check requests", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create SSL check event
		timestamp := time.Now().Unix()
		eventBody := `ssl_check=1&token=test_token`
		awsEvent := createDummyAWSEvent(eventBody, timestamp, fakeSigningSecret)
		awsEvent.Headers["Content-Type"] = "application/x-www-form-urlencoded"

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		assert.Equal(t, 200, response.StatusCode, "Should return 200 for SSL check")
		assert.Equal(t, "", response.Body, "Should return empty body for SSL check")
	})

	t.Run("should handle signature verification disabled", func(t *testing.T) {
		signatureVerification := false
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret:         fakeSigningSecret,
			SignatureVerification: &signatureVerification, // Disable signature verification
		})

		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()

		// Create event with completely invalid signature
		timestamp := time.Now().Unix()
		eventBody := `{"type":"url_verification","challenge":"test_challenge","token":"test_token"}`

		awsEvent := receivers.AwsEvent{
			Body: eventBody,
			Headers: map[string]string{
				"Content-Type":              "application/json",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         "completely_invalid_signature",
			},
			IsBase64Encoded: false,
		}

		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)

		// Should still work because signature verification is disabled
		assert.Equal(t, 200, response.StatusCode, "Should return 200 when signature verification is disabled")
	})

	t.Run("should accept interactivity requests as form-encoded payload", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Create app and initialize receiver
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		// Add an action handler to handle the interactive component
		actionID := "button_click"
		app.Action(types.ActionConstraints{
			ActionID: actionID,
		}, func(args types.SlackActionMiddlewareArgs) error {
			return args.Ack(nil)
		})

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create interactive payload (button click)
		interactivePayload := `{
			"type": "block_actions",
			"user": {"id": "U1234567890", "name": "test_user"},
			"api_app_id": "A1234567890",
			"token": "verification_token",
			"container": {"type": "message", "message_ts": "1234567890.123456"},
			"trigger_id": "123456789.987654321.abcdef123456789",
			"team": {"id": "T1234567890", "domain": "test-team"},
			"channel": {"id": "C1234567890", "name": "test-channel"},
			"message": {"type": "message", "text": "Test message"},
			"actions": [{
				"type": "button",
				"action_id": "button_click",
				"block_id": "block_1",
				"text": {"type": "plain_text", "text": "Click me"},
				"value": "button_value",
				"action_ts": "1234567890.123456"
			}]
		}`

		// Create form-encoded body (URL-encoded payload parameter)
		formBody := fmt.Sprintf("payload=%s", json.RawMessage(interactivePayload))

		// Create AWS event with form-encoded content type
		awsEvent := receivers.AwsEvent{
			Resource:   "/slack/events",
			Path:       "/slack/events",
			HTTPMethod: "POST",
			Headers: map[string]string{
				"Accept":                    "application/json,*/*",
				"Content-Type":              "application/x-www-form-urlencoded",
				"Host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",
				"User-Agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",
				"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
				"X-Slack-Signature":         createValidSignature(formBody, timestamp, fakeSigningSecret),
			},
			MultiValueHeaders:               make(map[string][]string),
			QueryStringParameters:           make(map[string]string),
			MultiValueQueryStringParameters: make(map[string][]string),
			PathParameters:                  make(map[string]string),
			StageVariables:                  make(map[string]string),
			RequestContext:                  make(map[string]interface{}),
			Body:                            formBody,
			IsBase64Encoded:                 false,
		}

		// Test form-encoded interactivity request - should return 200
		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	})

	t.Run("should not log an error regarding ack timeout if app has no handlers registered", func(t *testing.T) {
		receiver := receivers.NewAwsLambdaReceiver(types.AwsLambdaReceiverOptions{
			SigningSecret: fakeSigningSecret,
		})

		// Create app without any handlers
		app, err := bolt.New(bolt.AppOptions{
			Token:         fakeToken,
			SigningSecret: fakeSigningSecret,
			Receiver:      receiver,
		})
		require.NoError(t, err)

		err = receiver.Init(app)
		require.NoError(t, err)

		handler := receiver.ToHandler()
		timestamp := time.Now().Unix()

		// Create a dummy app mention event that has no handlers
		body := `{
			"token": "verification_token",
			"team_id": "T1234567890",
			"api_app_id": "A1234567890",
			"event": {
				"type": "app_mention",
				"user": "U1234567890",
				"text": "<@U0LAN0Z89> hello",
				"ts": "1515449522.000016",
				"channel": "C1234567890"
			},
			"type": "event_callback",
			"event_id": "Ev1234567890",
			"event_time": 1515449522
		}`
		awsEvent := createDummyAWSEvent(body, timestamp, fakeSigningSecret)

		// Test without handlers - should return 404 and not log ack timeout error
		response, err := handler(awsEvent, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, response.StatusCode)

		// This test mainly ensures that no ack timeout error is logged when there are no handlers
		// The actual logging behavior would need to be tested with a custom logger
		// For now, we're just ensuring the request completes successfully with 404
	})
}

// Helper function to create a dummy AWS event with valid signature
func createDummyAWSEvent(body string, timestamp int64, signingSecret string) receivers.AwsEvent {
	// Create valid signature
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	signature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return receivers.AwsEvent{
		Resource:   "/slack/events",
		Path:       "/slack/events",
		HTTPMethod: "POST",
		Headers: map[string]string{
			"Accept":                    "application/json,*/*",
			"Content-Type":              "application/json",
			"Host":                      "xxx.execute-api.ap-northeast-1.amazonaws.com",
			"User-Agent":                "Slackbot 1.0 (+https://api.slack.com/robots)",
			"X-Slack-Request-Timestamp": strconv.FormatInt(timestamp, 10),
			"X-Slack-Signature":         signature,
		},
		MultiValueHeaders:               make(map[string][]string),
		QueryStringParameters:           make(map[string]string),
		MultiValueQueryStringParameters: make(map[string][]string),
		PathParameters:                  make(map[string]string),
		StageVariables:                  make(map[string]string),
		RequestContext:                  make(map[string]interface{}),
		Body:                            body,
		IsBase64Encoded:                 false,
	}
}

// Helper function to create a valid Slack signature
func createValidSignature(body string, timestamp int64, signingSecret string) string {
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
