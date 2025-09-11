package test

import (
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	t.Run("CodedError interface", func(t *testing.T) {
		t.Run("should identify coded errors", func(t *testing.T) {
			err := bolt.NewAppInitializationError("test error")
			assert.True(t, bolt.IsCodedError(err))
			assert.Equal(t, bolt.AppInitializationErrorCode, err.Code())
		})

		t.Run("should convert regular errors to coded errors", func(t *testing.T) {
			regularErr := assert.AnError
			codedErr := bolt.AsCodedError(regularErr)
			assert.True(t, bolt.IsCodedError(codedErr))
			assert.Equal(t, bolt.UnknownErrorCode, codedErr.Code())
		})
	})

	t.Run("AppInitializationError", func(t *testing.T) {
		err := bolt.NewAppInitializationError("test message")
		assert.Equal(t, "test message", err.Error())
		assert.Equal(t, bolt.AppInitializationErrorCode, err.Code())
	})

	t.Run("AssistantInitializationError", func(t *testing.T) {
		err := bolt.NewAssistantInitializationError("assistant error")
		assert.Equal(t, "assistant error", err.Error())
		assert.Equal(t, bolt.AssistantInitializationErrorCode, err.Code())
	})

	t.Run("AssistantMissingPropertyError", func(t *testing.T) {
		err := bolt.NewAssistantMissingPropertyError("missing property")
		assert.Equal(t, "missing property", err.Error())
		assert.Equal(t, bolt.AssistantMissingPropertyErrorCode, err.Code())
	})

	t.Run("AuthorizationError", func(t *testing.T) {
		originalErr := assert.AnError
		err := bolt.NewAuthorizationError("auth failed", originalErr)
		assert.Equal(t, "auth failed", err.Error())
		assert.Equal(t, bolt.AuthorizationErrorCode, err.Code())
		assert.Equal(t, originalErr, err.Original())
	})

	t.Run("ContextMissingPropertyError", func(t *testing.T) {
		err := bolt.NewContextMissingPropertyError("botToken", "bot token is required")
		assert.Equal(t, "bot token is required", err.Error())
		assert.Equal(t, bolt.ContextMissingPropertyErrorCode, err.Code())
	})

	t.Run("ReceiverMultipleAckError", func(t *testing.T) {
		err := bolt.NewReceiverMultipleAckError()
		assert.Contains(t, err.Error(), "multiple times")
		assert.Equal(t, bolt.ReceiverMultipleAckErrorCode, err.Code())
	})

	t.Run("ReceiverAuthenticityError", func(t *testing.T) {
		err := bolt.NewReceiverAuthenticityError("invalid signature")
		assert.Equal(t, "invalid signature", err.Error())
		assert.Equal(t, bolt.ReceiverAuthenticityErrorCode, err.Code())
	})

	t.Run("MultipleListenerError", func(t *testing.T) {
		errors := []error{
			assert.AnError,
			bolt.NewAppInitializationError("test"),
		}
		err := bolt.NewMultipleListenerError(errors)
		assert.Contains(t, err.Error(), "Multiple errors")
		assert.Equal(t, bolt.MultipleListenerErrorCode, err.Code())
		assert.Equal(t, errors, err.Originals())
	})

	t.Run("WorkflowStepInitializationError", func(t *testing.T) {
		err := bolt.NewWorkflowStepInitializationError("workflow error")
		assert.Equal(t, "workflow error", err.Error())
		assert.Equal(t, bolt.WorkflowStepInitializationErrorCode, err.Code())
	})

	// Custom function errors removed for now
	// t.Run("CustomFunctionInitializationError", func(t *testing.T) {
	//	err := bolt.NewCustomFunctionInitializationError("function error")
	//	assert.Equal(t, "function error", err.Error())
	//	assert.Equal(t, bolt.CustomFunctionInitializationErrorCode, err.Code())
	// })
}

func TestErrorCodes(t *testing.T) {
	t.Run("should have correct error codes", func(t *testing.T) {
		assert.Equal(t, "slack_bolt_app_initialization_error", string(bolt.AppInitializationErrorCode))
		assert.Equal(t, "slack_bolt_assistant_initialization_error", string(bolt.AssistantInitializationErrorCode))
		assert.Equal(t, "slack_bolt_assistant_missing_property_error", string(bolt.AssistantMissingPropertyErrorCode))
		assert.Equal(t, "slack_bolt_authorization_error", string(bolt.AuthorizationErrorCode))
		assert.Equal(t, "slack_bolt_context_missing_property_error", string(bolt.ContextMissingPropertyErrorCode))
		assert.Equal(t, "slack_bolt_context_invalid_custom_property_error", string(bolt.InvalidCustomPropertyErrorCode))
		assert.Equal(t, "slack_bolt_receiver_ack_multiple_error", string(bolt.ReceiverMultipleAckErrorCode))
		assert.Equal(t, "slack_bolt_receiver_authenticity_error", string(bolt.ReceiverAuthenticityErrorCode))
		assert.Equal(t, "slack_bolt_multiple_listener_error", string(bolt.MultipleListenerErrorCode))
		assert.Equal(t, "slack_bolt_unknown_error", string(bolt.UnknownErrorCode))
		assert.Equal(t, "slack_bolt_workflow_step_initialization_error", string(bolt.WorkflowStepInitializationErrorCode))
		assert.Equal(t, "slack_bolt_custom_function_initialization_error", string(bolt.CustomFunctionInitializationErrorCode))
	})

	// Tests matching errors.spec.ts from JavaScript implementation
	t.Run("ErrorsSpecJS", func(t *testing.T) {
		t.Run("has errors matching codes", func(t *testing.T) {
			// Test that each error type has the expected error code
			errorMap := map[bolt.ErrorCode]bolt.CodedError{
				bolt.AppInitializationErrorCode:      bolt.NewAppInitializationError("test"),
				bolt.AuthorizationErrorCode:          bolt.NewAuthorizationError("auth failed", assert.AnError),
				bolt.ContextMissingPropertyErrorCode: bolt.NewContextMissingPropertyError("foo", "can't find foo"),
				bolt.ReceiverAuthenticityErrorCode:   bolt.NewReceiverAuthenticityError("authenticity failed"),
				// Note: Some constructors may not be available or have different signatures
			}

			for expectedCode, err := range errorMap {
				assert.Equal(t, expectedCode, err.Code(), "Error %T should have code %s", err, expectedCode)
			}
		})

		t.Run("wraps non-coded errors", func(t *testing.T) {
			regularErr := assert.AnError
			wrappedErr := bolt.AsCodedError(regularErr)

			// Should wrap as an error with UnknownErrorCode
			assert.True(t, bolt.IsCodedError(wrappedErr), "Should wrap as coded error")
			assert.Equal(t, bolt.UnknownErrorCode, wrappedErr.Code())
		})

		t.Run("passes coded errors through", func(t *testing.T) {
			originalErr := bolt.NewAppInitializationError("test")
			passedThroughErr := bolt.AsCodedError(originalErr)

			// Should be the same instance
			assert.Equal(t, originalErr, passedThroughErr, "Coded errors should pass through unchanged")
		})
	})
}
