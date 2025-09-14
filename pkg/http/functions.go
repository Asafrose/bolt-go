package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Asafrose/bolt-go/pkg/errors"
)

// ExtractRetryNumFromHTTPRequest extracts the retry number from the X-Slack-Retry-Num header
func ExtractRetryNumFromHTTPRequest(req *http.Request) *int {
	retryNumHeader := req.Header.Get("X-Slack-Retry-Num")
	if retryNumHeader == "" {
		return nil
	}

	retryNum, err := strconv.Atoi(retryNumHeader)
	if err != nil {
		return nil
	}

	return &retryNum
}

// ExtractRetryReasonFromHTTPRequest extracts the retry reason from the X-Slack-Retry-Reason header
func ExtractRetryReasonFromHTTPRequest(req *http.Request) *string {
	retryReason := req.Header.Get("X-Slack-Retry-Reason")
	if retryReason == "" {
		return nil
	}

	return &retryReason
}

// ParseHTTPRequestBody parses the HTTP request body based on content type
func ParseHTTPRequestBody(req *http.Request, rawBody []byte) (interface{}, error) {
	bodyStr := string(rawBody)
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		values, err := url.ParseQuery(bodyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse form data: %w", err)
		}

		// Check if there's a payload field (common in Slack requests)
		if payload := values.Get("payload"); payload != "" {
			var result interface{}
			if err := json.Unmarshal([]byte(payload), &result); err != nil {
				return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
			}
			return result, nil
		}

		// Return parsed form values as map
		result := make(map[string]interface{})
		for key, vals := range values {
			if len(vals) == 1 {
				result[key] = vals[0]
			} else {
				result[key] = vals
			}
		}
		return result, nil
	}

	// Default to JSON parsing
	var result interface{}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// GetHeader gets a header value and throws an error if missing
func GetHeader(req *http.Request, headerName string) (string, error) {
	value := req.Header.Get(headerName)
	if value == "" {
		return "", fmt.Errorf("header %s did not have the expected type (received empty, expected string)", headerName)
	}
	return value, nil
}

// BuildContentResponse builds an HTTP response with content
func BuildContentResponse(w http.ResponseWriter, body interface{}) {
	if body == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch v := body.(type) {
	case string:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(body)
	}
}

// BuildNoBodyResponse builds an HTTP response with no body
func BuildNoBodyResponse(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// BuildSSLCheckResponse builds a response for SSL check requests
func BuildSSLCheckResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// BuildUrlVerificationResponse builds a response for URL verification requests
func BuildUrlVerificationResponse(w http.ResponseWriter, body interface{}) {
	if bodyMap, ok := body.(map[string]interface{}); ok {
		if challenge, exists := bodyMap["challenge"]; exists {
			response := map[string]interface{}{
				"challenge": challenge,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
}

// DefaultDispatchErrorHandler handles dispatch errors
func DefaultDispatchErrorHandler(args DispatchErrorHandlerArgs) {
	if codedErr, ok := args.Error.(*errors.ReceiverMultipleAckError); ok {
		_ = codedErr // Use the error
		args.Logger.Error("Multiple ack error occurred")
		BuildNoBodyResponse(args.Response, http.StatusInternalServerError)
		return
	}

	if codedErr, ok := args.Error.(*errors.HTTPReceiverDeferredRequestError); ok {
		_ = codedErr // Use the error
		args.Logger.Info(fmt.Sprintf("Unhandled HTTP request (%s) made to %s", args.Request.Method, args.Request.URL.Path))
		BuildNoBodyResponse(args.Response, http.StatusNotFound)
		return
	}

	args.Logger.Error(fmt.Sprintf("An unexpected error occurred during a request (%s) made to %s", args.Request.Method, args.Request.URL.Path))
	args.Logger.Debug(fmt.Sprintf("Error details: %v", args.Error))
	BuildNoBodyResponse(args.Response, http.StatusInternalServerError)
}

// DefaultProcessEventErrorHandler handles process event errors
func DefaultProcessEventErrorHandler(args ProcessEventErrorHandlerArgs) bool {
	if codedErr, ok := args.Error.(*errors.ReceiverMultipleAckError); ok {
		_ = codedErr // Use the error
		args.Logger.Error("Multiple ack error occurred after ack() called in a listener")
		args.Logger.Debug(fmt.Sprintf("Error details: %v, storedResponse: %v", args.Error, args.StoredResponse))
		BuildNoBodyResponse(args.Response, http.StatusInternalServerError)
		return false
	}

	if codedErr, ok := args.Error.(*errors.AuthorizationError); ok {
		_ = codedErr // Use the error
		BuildNoBodyResponse(args.Response, http.StatusUnauthorized)
		return true
	}

	args.Logger.Error("An unhandled error occurred while Bolt processed an event")
	args.Logger.Debug(fmt.Sprintf("Error details: %v, storedResponse: %v", args.Error, args.StoredResponse))
	BuildNoBodyResponse(args.Response, http.StatusInternalServerError)
	return false
}

// DefaultUnhandledRequestHandler handles unhandled requests
func DefaultUnhandledRequestHandler(args UnhandledRequestHandlerArgs) {
	args.Logger.Error("An incoming event was not acknowledged within 3 seconds. Ensure that the ack() argument is called in a listener.")
	BuildNoBodyResponse(args.Response, http.StatusNotFound)
}

// DispatchErrorHandlerArgs contains arguments for dispatch error handlers
type DispatchErrorHandlerArgs struct {
	Error    error
	Logger   Logger
	Request  *http.Request
	Response http.ResponseWriter
}

// ProcessEventErrorHandlerArgs contains arguments for process event error handlers
type ProcessEventErrorHandlerArgs struct {
	Error          error
	Logger         Logger
	Request        *http.Request
	Response       http.ResponseWriter
	StoredResponse interface{}
}

// UnhandledRequestHandlerArgs contains arguments for unhandled request handlers
type UnhandledRequestHandlerArgs struct {
	Logger   Logger
	Request  *http.Request
	Response http.ResponseWriter
}

// Logger interface for logging
type Logger interface {
	Error(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}
