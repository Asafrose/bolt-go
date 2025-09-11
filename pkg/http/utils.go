package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/helpers"
)

// ExtractRetryNumFromHTTPRequest extracts the retry number from x-slack-retry-num header
func ExtractRetryNumFromHTTPRequest(req *http.Request) *int {
	retryNumHeaderValue := req.Header.Get("x-slack-retry-num")
	if retryNumHeaderValue == "" {
		return nil
	}

	retryNum, err := strconv.Atoi(retryNumHeaderValue)
	if err != nil {
		return nil
	}

	return &retryNum
}

// ExtractRetryReasonFromHTTPRequest extracts the retry reason from x-slack-retry-reason header
func ExtractRetryReasonFromHTTPRequest(req *http.Request) *string {
	retryReasonHeaderValue := req.Header.Get("x-slack-retry-reason")
	if retryReasonHeaderValue == "" {
		return nil
	}

	return &retryReasonHeaderValue
}

// ParseHTTPRequestBody parses the HTTP request body based on content type
func ParseHTTPRequestBody(req *http.Request, body []byte) (interface{}, error) {
	bodyAsString := string(body)
	contentType := req.Header.Get("content-type")

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		values, err := url.ParseQuery(bodyAsString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse form data: %w", err)
		}

		// Check if there's a payload field (Slack often sends JSON in a payload field)
		if payload := values.Get("payload"); payload != "" {
			var result interface{}
			if err := json.Unmarshal([]byte(payload), &result); err != nil {
				return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
			}
			return result, nil
		}

		// Return the parsed form values
		result := make(map[string]interface{})
		for key, values := range values {
			if len(values) == 1 {
				result[key] = values[0]
			} else {
				result[key] = values
			}
		}
		return result, nil
	}

	// Default to JSON parsing
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return result, nil
}

// GetHeader extracts a required header from the request
func GetHeader(req *http.Request, header string) (string, error) {
	value := req.Header.Get(header)
	if value == "" {
		return "", fmt.Errorf("Failed to verify authenticity: header %s did not have the expected type (received undefined, expected string)", header)
	}
	return value, nil
}

// RequestVerificationOptions represents options for request verification
type RequestVerificationOptions struct {
	Enabled       *bool
	SigningSecret string
	Logger        *slog.Logger
}

// ParseAndVerifyHTTPRequest parses and verifies an HTTP request
func ParseAndVerifyHTTPRequest(options RequestVerificationOptions, req *http.Request, body []byte) ([]byte, error) {
	// If verification is explicitly disabled, return the body immediately
	if options.Enabled != nil && !*options.Enabled {
		return body, nil
	}

	textBody := string(body)
	contentType := req.Header.Get("content-type")

	// SSL check requests don't require signature verification
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(textBody)
		if err == nil && values.Get("ssl_check") != "" {
			return body, nil
		}
	}

	// Extract required headers
	signature, err := GetHeader(req, "x-slack-signature")
	if err != nil {
		return nil, err
	}

	timestampStr, err := GetHeader(req, "x-slack-request-timestamp")
	if err != nil {
		return nil, err
	}

	// Parse timestamp for validation (used in VerifySlackSignature)
	_, err = strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to verify authenticity: invalid timestamp format")
	}

	// Verify the request
	if err := helpers.VerifySlackSignature(options.SigningSecret, signature, timestampStr, body); err != nil {
		return nil, err
	}

	return body, nil
}

// BuildNoBodyResponse builds an HTTP response with no body
func BuildNoBodyResponse(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// BuildURLVerificationResponse builds a URL verification response
func BuildURLVerificationResponse(w http.ResponseWriter, body map[string]interface{}) error {
	challenge, exists := body["challenge"]
	if !exists {
		return fmt.Errorf("challenge field not found in request body")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	response := map[string]interface{}{
		"challenge": challenge,
	}

	return json.NewEncoder(w).Encode(response)
}

// BuildSSLCheckResponse builds an SSL check response
func BuildSSLCheckResponse(w http.ResponseWriter) {
	w.WriteHeader(200)
}

// BuildContentResponse builds a content response
func BuildContentResponse(w http.ResponseWriter, body interface{}) error {
	if body == nil {
		w.WriteHeader(200)
		return nil
	}

	if str, ok := body.(string); ok {
		w.WriteHeader(200)
		_, err := w.Write([]byte(str))
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	return json.NewEncoder(w).Encode(body)
}

// ReceiverDispatchErrorHandlerArgs represents arguments for dispatch error handler
type ReceiverDispatchErrorHandlerArgs struct {
	Error    error
	Logger   *slog.Logger
	Request  *http.Request
	Response http.ResponseWriter
}

// DefaultDispatchErrorHandler is the default dispatch error handler
func DefaultDispatchErrorHandler(args ReceiverDispatchErrorHandlerArgs) {
	if codedError, ok := args.Error.(errors.CodedError); ok {
		if codedError.Code() == errors.HTTPReceiverDeferredRequestError {
			args.Logger.Info(fmt.Sprintf("Unhandled HTTP request (%s) made to %s", args.Request.Method, args.Request.URL.String()))
			args.Response.WriteHeader(404)
			return
		}
	}

	args.Logger.Error(fmt.Sprintf("An unexpected error occurred during a request (%s) made to %s", args.Request.Method, args.Request.URL.String()))
	args.Logger.Debug(fmt.Sprintf("Error details: %v", args.Error))
	args.Response.WriteHeader(500)
}

// ReceiverProcessEventErrorHandlerArgs represents arguments for process event error handler
type ReceiverProcessEventErrorHandlerArgs struct {
	Error          error
	Logger         *slog.Logger
	Request        *http.Request
	Response       http.ResponseWriter
	StoredResponse interface{}
}

// DefaultProcessEventErrorHandler is the default process event error handler
func DefaultProcessEventErrorHandler(args ReceiverProcessEventErrorHandlerArgs) bool {
	// Check if we can still write to the response
	// In Go, we can't easily check if headers have been sent, so we'll try to write

	if codedError, ok := args.Error.(errors.CodedError); ok {
		if codedError.Code() == errors.AuthorizationError {
			// Authorization function threw an exception, which means there is no valid installation data
			args.Response.WriteHeader(401)
			return true
		}

		if codedError.Code() == errors.ReceiverMultipleAckError {
			args.Logger.Error("An unhandled error occurred after ack() called in a listener")
			args.Logger.Debug(fmt.Sprintf("Error details: %v, storedResponse: %v", args.Error, args.StoredResponse))
			args.Response.WriteHeader(500)
			return false
		}
	}

	args.Logger.Error("An unhandled error occurred while Bolt processed an event")
	args.Logger.Debug(fmt.Sprintf("Error details: %v, storedResponse: %v", args.Error, args.StoredResponse))
	args.Response.WriteHeader(500)
	return false
}

// ReceiverUnhandledRequestHandlerArgs represents arguments for unhandled request handler
type ReceiverUnhandledRequestHandlerArgs struct {
	Logger   *slog.Logger
	Request  *http.Request
	Response http.ResponseWriter
}

// DefaultUnhandledRequestHandler is the default unhandled request handler
func DefaultUnhandledRequestHandler(args ReceiverUnhandledRequestHandlerArgs) {
	args.Logger.Error("An incoming event was not acknowledged within 3 seconds. Ensure that the ack() argument is called in a listener.")

	// Set the status code and end the response
	args.Response.WriteHeader(404) // Not Found
}

// ReadRequestBody reads the request body and returns it as bytes
func ReadRequestBody(req *http.Request) ([]byte, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer req.Body.Close()
	return body, nil
}
