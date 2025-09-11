package receivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/types"
)

// AwsEvent represents an AWS Lambda event (supports both v1 and v2)
type AwsEvent struct {
	// Common fields
	Body            string                 `json:"body"`
	Headers         map[string]string      `json:"headers"`
	IsBase64Encoded bool                   `json:"isBase64Encoded"`
	RequestContext  map[string]interface{} `json:"requestContext"`

	// V1 fields
	HTTPMethod                      string              `json:"httpMethod,omitempty"`
	Path                            string              `json:"path,omitempty"`
	Resource                        string              `json:"resource,omitempty"`
	PathParameters                  map[string]string   `json:"pathParameters"`
	QueryStringParameters           map[string]string   `json:"queryStringParameters"`
	MultiValueHeaders               map[string][]string `json:"multiValueHeaders"`
	MultiValueQueryStringParameters map[string][]string `json:"multiValueQueryStringParameters"`
	StageVariables                  map[string]string   `json:"stageVariables"`

	// V2 fields
	Version        string   `json:"version,omitempty"`
	RouteKey       string   `json:"routeKey,omitempty"`
	RawPath        string   `json:"rawPath,omitempty"`
	RawQueryString string   `json:"rawQueryString,omitempty"`
	Cookies        []string `json:"cookies,omitempty"`
}

// AwsResponse represents an AWS Lambda response
type AwsResponse struct {
	StatusCode        int                 `json:"statusCode"`
	Headers           map[string]string   `json:"headers,omitempty"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders,omitempty"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded,omitempty"`
}

// AwsCallback represents the AWS Lambda callback function
type AwsCallback func(error interface{}, result interface{})

// AwsHandler represents the AWS Lambda handler function signature
type AwsHandler func(event AwsEvent, context interface{}, callback AwsCallback) (AwsResponse, error)

// AwsLambdaReceiver handles AWS Lambda requests from Slack
type AwsLambdaReceiver struct {
	signingSecret                 string
	logger                        *slog.Logger
	processBeforeResponse         bool
	signatureVerification         bool
	unhandledRequestTimeoutMillis int
	customProperties              map[string]interface{}

	app types.App
}

// NewAwsLambdaReceiver creates a new AWS Lambda receiver
func NewAwsLambdaReceiver(options types.AwsLambdaReceiverOptions) *AwsLambdaReceiver {
	// Default signature verification to true
	signatureVerification := true
	if options.SignatureVerification != nil {
		signatureVerification = *options.SignatureVerification
	}

	receiver := &AwsLambdaReceiver{
		signingSecret:                 options.SigningSecret,
		processBeforeResponse:         options.ProcessBeforeResponse,
		unhandledRequestTimeoutMillis: 3001, // default
		signatureVerification:         signatureVerification,
		customProperties:              options.CustomProperties,
	}

	if options.Logger != nil {
		if logger, ok := options.Logger.(*slog.Logger); ok {
			receiver.logger = logger
		}
	}

	if receiver.logger == nil {
		receiver.logger = slog.Default()
	}

	return receiver
}

// Init initializes the receiver with the app
func (r *AwsLambdaReceiver) Init(app types.App) error {
	r.app = app
	return nil
}

// Start starts the receiver (no-op for Lambda)
func (r *AwsLambdaReceiver) Start(ctx context.Context) error {
	if r.app == nil {
		return errors.NewAppInitializationError("receiver not initialized")
	}
	// Lambda doesn't have a persistent server to start
	return nil
}

// Stop stops the receiver (no-op for Lambda)
func (r *AwsLambdaReceiver) Stop(ctx context.Context) error {
	// Lambda doesn't have a persistent server to stop
	return nil
}

// APIGatewayProxyEvent represents an AWS API Gateway proxy event
type APIGatewayProxyEvent struct {
	HTTPMethod            string            `json:"httpMethod"`
	Path                  string            `json:"path"`
	PathParameters        map[string]string `json:"pathParameters"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	Headers               map[string]string `json:"headers"`
	Body                  string            `json:"body"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
	RequestContext        RequestContext    `json:"requestContext"`
}

// RequestContext represents the AWS API Gateway request context
type RequestContext struct {
	AccountID    string `json:"accountId"`
	APIID        string `json:"apiId"`
	HTTPMethod   string `json:"httpMethod"`
	RequestID    string `json:"requestId"`
	ResourcePath string `json:"resourcePath"`
	Stage        string `json:"stage"`
}

// APIGatewayProxyResponse represents an AWS API Gateway proxy response
type APIGatewayProxyResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// HandleLambdaEvent processes an AWS Lambda event
func (r *AwsLambdaReceiver) HandleLambdaEvent(ctx context.Context, event APIGatewayProxyEvent) (APIGatewayProxyResponse, error) {
	if r.app == nil {
		return r.createErrorResponse(500, "Receiver not initialized"), nil
	}

	// Only handle POST requests
	if event.HTTPMethod != "POST" {
		return r.createErrorResponse(405, "Method not allowed"), nil
	}

	// Parse the body
	var bodyBytes []byte
	if event.IsBase64Encoded {
		// Decode base64 encoded body
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return r.createErrorResponse(400, "Invalid base64 body"), nil
		}
		bodyBytes = decoded
	} else {
		bodyBytes = []byte(event.Body)
	}

	// Handle URL verification
	if r.isURLVerification(bodyBytes) {
		return r.handleURLVerification(bodyBytes)
	}

	// Verify signature if enabled
	if r.signatureVerification {
		if err := r.verifySignature(event.Headers, bodyBytes); err != nil {
			r.logger.Error("Signature verification failed", "error", err)
			return r.createErrorResponse(401, "Unauthorized"), nil
		}
	}

	// Convert headers to the format expected by ReceiverEvent
	headers := make(map[string]string)
	for k, v := range event.Headers {
		headers[strings.ToLower(k)] = v
	}

	// Handle form-encoded data (for slash commands and interactive components)
	if contentType, exists := headers["content-type"]; exists &&
		strings.Contains(contentType, "application/x-www-form-urlencoded") {
		bodyBytes = r.parseFormData(bodyBytes)
	}

	// Create receiver event
	receiverEvent := types.ReceiverEvent{
		Body:    bodyBytes,
		Headers: headers,
		Ack: func(response interface{}) error {
			// For Lambda, ack is handled by returning the response
			return nil
		},
	}

	// Process the event
	if r.processBeforeResponse {
		// Process synchronously
		err := r.app.ProcessEvent(ctx, receiverEvent)
		if err != nil {
			r.logger.Error("Error processing event", "error", err)
			return r.createErrorResponse(500, "Internal server error"), nil
		}
		return r.createSuccessResponse(), nil
	} else {
		// Process asynchronously (fire and forget)
		go func() {
			err := r.app.ProcessEvent(context.Background(), receiverEvent)
			if err != nil {
				r.logger.Error("Error processing event asynchronously", "error", err)
			}
		}()
		return r.createSuccessResponse(), nil
	}
}

// isURLVerification checks if the request is a URL verification request
func (r *AwsLambdaReceiver) isURLVerification(body []byte) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}

	eventType, exists := parsed["type"]
	if !exists {
		return false
	}

	return eventType == "url_verification"
}

// handleURLVerification handles URL verification requests
func (r *AwsLambdaReceiver) handleURLVerification(body []byte) (APIGatewayProxyResponse, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return r.createErrorResponse(400, "Invalid JSON"), nil
	}

	challenge, exists := parsed["challenge"]
	if !exists {
		return r.createErrorResponse(400, "Missing challenge"), nil
	}

	challengeStr, ok := challenge.(string)
	if !ok {
		return r.createErrorResponse(400, "Invalid challenge type"), nil
	}

	return APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: challengeStr,
	}, nil
}

// verifySignature verifies the Slack request signature
func (r *AwsLambdaReceiver) verifySignature(headers map[string]string, body []byte) error {
	// Get signature and timestamp from headers
	signature := ""
	timestamp := ""

	for k, v := range headers {
		switch strings.ToLower(k) {
		case "x-slack-signature":
			signature = v
		case "x-slack-request-timestamp":
			timestamp = v
		}
	}

	if signature == "" || timestamp == "" {
		return fmt.Errorf("missing signature or timestamp headers")
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	// Check if request is too old (5 minutes)
	fiveMinutesAgo := time.Now().Unix() - 300
	if ts < fiveMinutesAgo {
		return fmt.Errorf("request timestamp too old")
	}

	// Parse signature
	parts := strings.Split(signature, "=")
	if len(parts) != 2 {
		return fmt.Errorf("invalid signature format")
	}
	version := parts[0]
	hash := parts[1]

	// Create HMAC
	h := hmac.New(sha256.New, []byte(r.signingSecret))
	h.Write([]byte(fmt.Sprintf("%s:%s:%s", version, timestamp, body)))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	// Compare hashes using constant time comparison
	if !hmac.Equal([]byte(hash), []byte(expectedHash)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// parseFormData parses form-encoded data and converts it to JSON
func (r *AwsLambdaReceiver) parseFormData(body []byte) []byte {
	bodyStr := string(body)

	// Check if this is a payload parameter (for interactive components)
	if strings.HasPrefix(bodyStr, "payload=") {
		// Extract the payload value
		payloadStart := len("payload=")
		if len(bodyStr) > payloadStart {
			// URL decode the payload
			payload := bodyStr[payloadStart:]
			// Simple URL decoding for common cases
			payload = strings.ReplaceAll(payload, "%22", "\"")
			payload = strings.ReplaceAll(payload, "%7B", "{")
			payload = strings.ReplaceAll(payload, "%7D", "}")
			payload = strings.ReplaceAll(payload, "%3A", ":")
			payload = strings.ReplaceAll(payload, "%2C", ",")
			payload = strings.ReplaceAll(payload, "%5B", "[")
			payload = strings.ReplaceAll(payload, "%5D", "]")
			payload = strings.ReplaceAll(payload, "+", " ")

			return []byte(payload)
		}
	}

	// For slash commands, parse form data into JSON
	if strings.Contains(bodyStr, "=") && strings.Contains(bodyStr, "&") {
		return r.convertFormToJSON(bodyStr)
	}

	return body
}

// convertFormToJSON converts form data to JSON format
func (r *AwsLambdaReceiver) convertFormToJSON(formData string) []byte {
	result := make(map[string]string)

	pairs := strings.Split(formData, "&")
	for _, pair := range pairs {
		if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				// Simple URL decoding
				value = strings.ReplaceAll(value, "+", " ")
				value = strings.ReplaceAll(value, "%20", " ")
				result[key] = value
			}
		}
	}

	jsonBytes, _ := json.Marshal(result)
	return jsonBytes
}

// createSuccessResponse creates a successful Lambda response
func (r *AwsLambdaReceiver) createSuccessResponse() APIGatewayProxyResponse {
	return APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"ok":true}`,
	}
}

// createErrorResponse creates an error Lambda response
func (r *AwsLambdaReceiver) createErrorResponse(statusCode int, message string) APIGatewayProxyResponse {
	errorBody := map[string]string{
		"error": message,
	}

	bodyBytes, _ := json.Marshal(errorBody)

	return APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyBytes),
	}
}

// ToHandler converts the receiver to an AWS Lambda handler
func (r *AwsLambdaReceiver) ToHandler() AwsHandler {
	return func(awsEvent AwsEvent, awsContext interface{}, awsCallback AwsCallback) (AwsResponse, error) {
		r.logger.Debug("AWS event", "event", awsEvent)

		rawBody := r.getRawBody(awsEvent)

		// Parse request body
		body := r.parseRequestBody(rawBody, r.getHeaderValue(awsEvent.Headers, "Content-Type"))

		// Handle SSL check (for Slash Commands)
		if r.isSSLCheck(body) {
			return AwsResponse{StatusCode: 200, Body: ""}, nil
		}

		// Handle signature verification
		if r.signatureVerification {
			signature := r.getHeaderValue(awsEvent.Headers, "X-Slack-Signature")
			tsStr := r.getHeaderValue(awsEvent.Headers, "X-Slack-Request-Timestamp")
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				return AwsResponse{StatusCode: 401, Body: ""}, nil
			}

			if !r.isValidRequestSignature(rawBody, signature, ts) {
				return AwsResponse{StatusCode: 401, Body: ""}, nil
			}
		}

		// Handle URL verification (Events API)
		if r.isURLVerificationFromMap(body) {
			if challenge, ok := body["challenge"].(string); ok {
				response := map[string]string{"challenge": challenge}
				responseBody, _ := json.Marshal(response)
				return AwsResponse{
					StatusCode: 200,
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       string(responseBody),
				}, nil
			}
		}

		// Process the event through the app
		isAcknowledged := false

		// Convert the parsed body back to JSON bytes for the ReceiverEvent
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			r.logger.Error("Error marshaling body", "error", err)
			return AwsResponse{StatusCode: 500, Body: "Internal Server Error"}, nil
		}

		receiverEvent := types.ReceiverEvent{
			Body:    bodyBytes,
			Headers: awsEvent.Headers,
			Ack: func(response interface{}) error {
				isAcknowledged = true
				return nil
			},
		}

		// Process the event
		ctx := context.Background()
		if err := r.app.ProcessEvent(ctx, receiverEvent); err != nil {
			r.logger.Error("Error processing event", "error", err)
			return AwsResponse{StatusCode: 500, Body: "Internal Server Error"}, nil
		}

		// Return appropriate response
		if isAcknowledged {
			return AwsResponse{StatusCode: 200, Body: ""}, nil
		} else {
			return AwsResponse{StatusCode: 404, Body: "Not Found"}, nil
		}
	}
}

// getRawBody extracts the raw body from AWS event
func (r *AwsLambdaReceiver) getRawBody(awsEvent AwsEvent) string {
	if awsEvent.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(awsEvent.Body)
		if err != nil {
			r.logger.Error("Failed to decode base64 body", "error", err)
			return awsEvent.Body // Return original if decoding fails
		}
		return string(decoded)
	}
	return awsEvent.Body
}

// parseRequestBody parses the request body based on content type
func (r *AwsLambdaReceiver) parseRequestBody(rawBody, contentType string) map[string]interface{} {
	result := make(map[string]interface{})

	if strings.Contains(contentType, "application/json") {
		json.Unmarshal([]byte(rawBody), &result)
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		values, err := url.ParseQuery(rawBody)
		if err == nil {
			// Check if there's a payload field (common for Slack interactive components)
			if payloadSlice, exists := values["payload"]; exists && len(payloadSlice) == 1 {
				// Parse the payload as JSON
				var payloadResult map[string]interface{}
				if err := json.Unmarshal([]byte(payloadSlice[0]), &payloadResult); err == nil {
					return payloadResult
				}
			}

			// Otherwise, parse as regular form data
			for key, valueSlice := range values {
				if len(valueSlice) == 1 {
					result[key] = valueSlice[0]
				} else {
					result[key] = valueSlice
				}
			}
		}
	}

	return result
}

// getHeaderValue gets a header value (case-insensitive)
func (r *AwsLambdaReceiver) getHeaderValue(headers map[string]string, key string) string {
	for headerKey, headerValue := range headers {
		if strings.EqualFold(headerKey, key) {
			return headerValue
		}
	}
	return ""
}

// isSSLCheck checks if this is an SSL check request
func (r *AwsLambdaReceiver) isSSLCheck(body map[string]interface{}) bool {
	if sslCheck, exists := body["ssl_check"]; exists {
		return sslCheck != nil
	}
	return false
}

// isURLVerificationFromMap checks if this is a URL verification request from parsed body
func (r *AwsLambdaReceiver) isURLVerificationFromMap(body map[string]interface{}) bool {
	if eventType, exists := body["type"]; exists {
		if typeStr, ok := eventType.(string); ok {
			return typeStr == "url_verification"
		}
	}
	return false
}

// isValidRequestSignature validates the request signature
func (r *AwsLambdaReceiver) isValidRequestSignature(rawBody, signature string, timestamp int64) bool {
	// Check if timestamp is too old (more than 5 minutes)
	now := time.Now().Unix()
	if now-timestamp > 300 {
		return false
	}

	// Create the signature base string
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, rawBody)

	// Calculate the expected signature
	mac := hmac.New(sha256.New, []byte(r.signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures (constant time comparison)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
