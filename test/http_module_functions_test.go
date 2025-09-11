package test

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Asafrose/bolt-go/pkg/errors"
	httputils "github.com/Asafrose/bolt-go/pkg/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestHeaderExtraction(t *testing.T) {
	t.Run("extractRetryNumFromHTTPRequest", func(t *testing.T) {
		t.Run("should work when the header does not exist", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			result := httputils.ExtractRetryNumFromHTTPRequest(req)
			assert.Nil(t, result)
		})

		t.Run("should parse a single value header", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("x-slack-retry-num", "2")
			result := httputils.ExtractRetryNumFromHTTPRequest(req)
			require.NotNil(t, result)
			assert.Equal(t, 2, *result)
		})

		t.Run("should parse an array of value headers", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Add("x-slack-retry-num", "2")
			req.Header.Add("x-slack-retry-num", "3") // Second value should be ignored
			result := httputils.ExtractRetryNumFromHTTPRequest(req)
			require.NotNil(t, result)
			assert.Equal(t, 2, *result) // Should get the first value
		})
	})

	t.Run("extractRetryReasonFromHTTPRequest", func(t *testing.T) {
		t.Run("should work when the header does not exist", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			result := httputils.ExtractRetryReasonFromHTTPRequest(req)
			assert.Nil(t, result)
		})

		t.Run("should parse a valid header", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("x-slack-retry-reason", "timeout")
			result := httputils.ExtractRetryReasonFromHTTPRequest(req)
			require.NotNil(t, result)
			assert.Equal(t, "timeout", *result)
		})

		t.Run("should parse an array of value headers", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Add("x-slack-retry-reason", "timeout")
			req.Header.Add("x-slack-retry-reason", "rate_limited") // Second value should be ignored
			result := httputils.ExtractRetryReasonFromHTTPRequest(req)
			require.NotNil(t, result)
			assert.Equal(t, "timeout", *result) // Should get the first value
		})
	})
}

func TestHTTPRequestParsingAndVerification(t *testing.T) {
	t.Run("parseHTTPRequestBody", func(t *testing.T) {
		t.Run("should parse a JSON request body", func(t *testing.T) {
			jsonBody := `{"foo":"bar"}`
			req := httptest.NewRequest("POST", "/", strings.NewReader(jsonBody))
			req.Header.Set("content-type", "application/json")

			result, err := httputils.ParseHTTPRequestBody(req, []byte(jsonBody))
			assert.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "bar", resultMap["foo"])
		})

		t.Run("should parse a form request body", func(t *testing.T) {
			payload := `{"foo":"bar"}`
			formBody := fmt.Sprintf("payload=%s", url.QueryEscape(payload))
			req := httptest.NewRequest("POST", "/", strings.NewReader(formBody))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			result, err := httputils.ParseHTTPRequestBody(req, []byte(formBody))
			assert.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "bar", resultMap["foo"])
		})
	})

	t.Run("getHeader", func(t *testing.T) {
		t.Run("should throw an exception when parsing a missing header", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			_, err := httputils.GetHeader(req, "Cookie")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Failed to verify authenticity")
			assert.Contains(t, err.Error(), "Cookie")
		})

		t.Run("should parse a valid header", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Cookie", "foo=bar")
			result, err := httputils.GetHeader(req, "Cookie")
			assert.NoError(t, err)
			assert.Equal(t, "foo=bar", result)
		})
	})

	t.Run("parseAndVerifyHTTPRequest", func(t *testing.T) {
		t.Run("should parse a JSON request body", func(t *testing.T) {
			signingSecret := "secret"
			timestamp := time.Now().Unix()
			rawBody := `{"foo":"bar"}`

			// Create valid signature
			mac := hmac.New(sha256.New, []byte(signingSecret))
			mac.Write([]byte(fmt.Sprintf("v0:%d:%s", timestamp, rawBody)))
			signature := fmt.Sprintf("v0=%x", mac.Sum(nil))

			req := httptest.NewRequest("POST", "/", strings.NewReader(rawBody))
			req.Header.Set("content-type", "application/json")
			req.Header.Set("x-slack-signature", signature)
			req.Header.Set("x-slack-request-timestamp", strconv.FormatInt(timestamp, 10))

			options := httputils.RequestVerificationOptions{
				SigningSecret: signingSecret,
			}

			result, err := httputils.ParseAndVerifyHTTPRequest(options, req, []byte(rawBody))
			assert.NoError(t, err)
			assert.Equal(t, []byte(rawBody), result)
		})

		t.Run("should detect an invalid timestamp", func(t *testing.T) {
			signingSecret := "secret"
			timestamp := time.Now().Unix() - 600 // 10 minutes ago
			rawBody := `{"foo":"bar"}`

			// Create signature with old timestamp
			mac := hmac.New(sha256.New, []byte(signingSecret))
			mac.Write([]byte(fmt.Sprintf("v0:%d:%s", timestamp, rawBody)))
			signature := fmt.Sprintf("v0=%x", mac.Sum(nil))

			req := httptest.NewRequest("POST", "/", strings.NewReader(rawBody))
			req.Header.Set("content-type", "application/json")
			req.Header.Set("x-slack-signature", signature)
			req.Header.Set("x-slack-request-timestamp", strconv.FormatInt(timestamp, 10))

			options := httputils.RequestVerificationOptions{
				SigningSecret: signingSecret,
			}

			_, err := httputils.ParseAndVerifyHTTPRequest(options, req, []byte(rawBody))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "timestamp")
		})

		t.Run("should detect an invalid signature", func(t *testing.T) {
			signingSecret := "secret"
			timestamp := time.Now().Unix()
			rawBody := `{"foo":"bar"}`

			req := httptest.NewRequest("POST", "/", strings.NewReader(rawBody))
			req.Header.Set("content-type", "application/json")
			req.Header.Set("x-slack-signature", "v0=invalid-signature")
			req.Header.Set("x-slack-request-timestamp", strconv.FormatInt(timestamp, 10))

			options := httputils.RequestVerificationOptions{
				SigningSecret: signingSecret,
			}

			_, err := httputils.ParseAndVerifyHTTPRequest(options, req, []byte(rawBody))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "signature")
		})

		t.Run("should parse a ssl_check request body without signature verification", func(t *testing.T) {
			signingSecret := "secret"
			rawBody := "ssl_check=1&token=legacy-fixed-verification-token"

			req := httptest.NewRequest("POST", "/", strings.NewReader(rawBody))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			options := httputils.RequestVerificationOptions{
				SigningSecret: signingSecret,
			}

			result, err := httputils.ParseAndVerifyHTTPRequest(options, req, []byte(rawBody))
			assert.NoError(t, err)
			assert.Equal(t, []byte(rawBody), result)
		})

		t.Run("should detect invalid signature for application/x-www-form-urlencoded body", func(t *testing.T) {
			signingSecret := "secret"
			rawBody := "payload={}"
			timestamp := time.Now().Unix()

			req := httptest.NewRequest("POST", "/", strings.NewReader(rawBody))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")
			req.Header.Set("x-slack-signature", "v0=invalid-signature")
			req.Header.Set("x-slack-request-timestamp", strconv.FormatInt(timestamp, 10))

			options := httputils.RequestVerificationOptions{
				SigningSecret: signingSecret,
			}

			_, err := httputils.ParseAndVerifyHTTPRequest(options, req, []byte(rawBody))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "signature")
		})
	})
}

func TestHTTPResponseBuilderMethods(t *testing.T) {
	t.Run("should have buildContentResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := httputils.BuildContentResponse(w, "OK")
		assert.NoError(t, err)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("should have buildNoBodyResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		httputils.BuildNoBodyResponse(w, 500)
		assert.Equal(t, 500, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("should have buildSSLCheckResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		httputils.BuildSSLCheckResponse(w)
		assert.Equal(t, 200, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("should have buildUrlVerificationResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]interface{}{
			"challenge": "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
		}
		err := httputils.BuildURLVerificationResponse(w, body)
		assert.NoError(t, err)
		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P")
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
}

func TestErrorHandlersForEventProcessing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("defaultDispatchErrorHandler", func(t *testing.T) {
		t.Run("should properly handle ReceiverMultipleAckError", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			w := httptest.NewRecorder()

			args := httputils.ReceiverDispatchErrorHandlerArgs{
				Error:    errors.NewReceiverMultipleAckError(),
				Logger:   logger,
				Request:  req,
				Response: w,
			}

			httputils.DefaultDispatchErrorHandler(args)
			assert.Equal(t, 500, w.Code)
		})

		t.Run("should properly handle HTTPReceiverDeferredRequestError", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			w := httptest.NewRecorder()

			args := httputils.ReceiverDispatchErrorHandlerArgs{
				Error:    errors.NewHTTPReceiverDeferredRequestError("msg", req, w),
				Logger:   logger,
				Request:  req,
				Response: w,
			}

			httputils.DefaultDispatchErrorHandler(args)
			assert.Equal(t, 404, w.Code)
		})
	})

	t.Run("defaultProcessEventErrorHandler", func(t *testing.T) {
		t.Run("should properly handle ReceiverMultipleAckError", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			w := httptest.NewRecorder()

			args := httputils.ReceiverProcessEventErrorHandlerArgs{
				Error:          errors.NewReceiverMultipleAckError(),
				StoredResponse: nil,
				Logger:         logger,
				Request:        req,
				Response:       w,
			}

			result := httputils.DefaultProcessEventErrorHandler(args)
			assert.False(t, result)
			assert.Equal(t, 500, w.Code)
		})

		t.Run("should properly handle AuthorizationError", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			w := httptest.NewRecorder()

			args := httputils.ReceiverProcessEventErrorHandlerArgs{
				Error:          errors.NewAuthorizationError("msg", fmt.Errorf("original error")),
				StoredResponse: nil,
				Logger:         logger,
				Request:        req,
				Response:       w,
			}

			result := httputils.DefaultProcessEventErrorHandler(args)
			assert.True(t, result)
			assert.Equal(t, 401, w.Code)
		})
	})

	t.Run("defaultUnhandledRequestHandler", func(t *testing.T) {
		t.Run("should properly execute", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			w := httptest.NewRecorder()

			args := httputils.ReceiverUnhandledRequestHandlerArgs{
				Logger:   logger,
				Request:  req,
				Response: w,
			}

			httputils.DefaultUnhandledRequestHandler(args)
			assert.Equal(t, 404, w.Code)
		})
	})
}

func TestAdditionalHTTPUtilities(t *testing.T) {
	t.Run("readRequestBody", func(t *testing.T) {
		t.Run("should read request body successfully", func(t *testing.T) {
			body := "test body content"
			req := httptest.NewRequest("POST", "/", strings.NewReader(body))

			result, err := httputils.ReadRequestBody(req)
			assert.NoError(t, err)
			assert.Equal(t, []byte(body), result)
		})

		t.Run("should handle empty body", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)

			result, err := httputils.ReadRequestBody(req)
			assert.NoError(t, err)
			assert.Empty(t, result)
		})
	})

	t.Run("buildContentResponse with different types", func(t *testing.T) {
		t.Run("should handle nil body", func(t *testing.T) {
			w := httptest.NewRecorder()
			err := httputils.BuildContentResponse(w, nil)
			assert.NoError(t, err)
			assert.Equal(t, 200, w.Code)
			assert.Empty(t, w.Body.String())
		})

		t.Run("should handle string body", func(t *testing.T) {
			w := httptest.NewRecorder()
			err := httputils.BuildContentResponse(w, "test string")
			assert.NoError(t, err)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "test string", w.Body.String())
		})

		t.Run("should handle JSON body", func(t *testing.T) {
			w := httptest.NewRecorder()
			body := map[string]interface{}{
				"message": "hello",
				"count":   42,
			}
			err := httputils.BuildContentResponse(w, body)
			assert.NoError(t, err)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Contains(t, w.Body.String(), "hello")
			assert.Contains(t, w.Body.String(), "42")
		})
	})

	t.Run("parseHTTPRequestBody with edge cases", func(t *testing.T) {
		t.Run("should handle form data without payload field", func(t *testing.T) {
			formBody := "field1=value1&field2=value2"
			req := httptest.NewRequest("POST", "/", strings.NewReader(formBody))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			result, err := httputils.ParseHTTPRequestBody(req, []byte(formBody))
			assert.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "value1", resultMap["field1"])
			assert.Equal(t, "value2", resultMap["field2"])
		})

		t.Run("should handle invalid JSON", func(t *testing.T) {
			invalidJSON := `{"invalid": json}`
			req := httptest.NewRequest("POST", "/", strings.NewReader(invalidJSON))
			req.Header.Set("content-type", "application/json")

			_, err := httputils.ParseHTTPRequestBody(req, []byte(invalidJSON))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse JSON")
		})
	})
}
