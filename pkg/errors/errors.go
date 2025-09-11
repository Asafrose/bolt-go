package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents error codes used throughout the framework
type ErrorCode string

const (
	AppInitializationErrorCode ErrorCode = "slack_bolt_app_initialization_error"

	AssistantInitializationErrorCode  ErrorCode = "slack_bolt_assistant_initialization_error"
	AssistantMissingPropertyErrorCode ErrorCode = "slack_bolt_assistant_missing_property_error"

	AuthorizationErrorCode ErrorCode = "slack_bolt_authorization_error"

	ContextMissingPropertyErrorCode ErrorCode = "slack_bolt_context_missing_property_error"
	InvalidCustomPropertyErrorCode  ErrorCode = "slack_bolt_context_invalid_custom_property_error"

	CustomRouteInitializationError ErrorCode = "slack_bolt_custom_route_initialization_error"

	ReceiverMultipleAckErrorCode   ErrorCode = "slack_bolt_receiver_ack_multiple_error"
	ReceiverAuthenticityErrorCode  ErrorCode = "slack_bolt_receiver_authenticity_error"
	ReceiverInconsistentStateError ErrorCode = "slack_bolt_receiver_inconsistent_state_error"

	MultipleListenerErrorCode ErrorCode = "slack_bolt_multiple_listener_error"

	HTTPReceiverDeferredRequestErrorCode ErrorCode = "slack_bolt_http_receiver_deferred_request_error"

	UnknownError ErrorCode = "slack_bolt_unknown_error"

	EventProcessingError ErrorCode = "slack_bolt_event_processing_error"

	WorkflowStepInitializationErrorCode ErrorCode = "slack_bolt_workflow_step_initialization_error"

	CustomFunctionInitializationErrorCode  ErrorCode = "slack_bolt_custom_function_initialization_error"
	CustomFunctionCompleteSuccessErrorCode ErrorCode = "slack_bolt_custom_function_complete_success_error"
	CustomFunctionCompleteFailErrorCode    ErrorCode = "slack_bolt_custom_function_complete_fail_error"
)

// CodedError represents an error with a specific error code
type CodedError interface {
	error
	Code() ErrorCode
	Original() error
	Originals() []error
}

// BaseError implements CodedError interface
type BaseError struct {
	code      ErrorCode
	message   string
	original  error
	originals []error
}

func (e BaseError) Error() string {
	return e.message
}

func (e BaseError) Code() ErrorCode {
	return e.code
}

func (e BaseError) Original() error {
	return e.original
}

func (e BaseError) Originals() []error {
	return e.originals
}

// NewBaseError creates a new BaseError
func NewBaseError(code ErrorCode, message string) *BaseError {
	return &BaseError{
		code:    code,
		message: message,
	}
}

// NewBaseErrorWithOriginal creates a new BaseError with an original error
func NewBaseErrorWithOriginal(code ErrorCode, message string, original error) *BaseError {
	return &BaseError{
		code:     code,
		message:  message,
		original: original,
	}
}

// IsCodedError checks if an error implements CodedError
func IsCodedError(err error) bool {
	var codedErr CodedError
	return errors.As(err, &codedErr)
}

// AsCodedError converts an error to a CodedError
func AsCodedError(err error) CodedError {
	var codedErr CodedError
	if errors.As(err, &codedErr) {
		return codedErr
	}
	return NewBaseErrorWithOriginal(UnknownError, err.Error(), err)
}

// Specific error types

// AppInitializationErrorType represents an app initialization error
type AppInitializationError struct {
	*BaseError
}

// NewAppInitializationError creates a new AppInitializationError
func NewAppInitializationError(message string) *AppInitializationError {
	return &AppInitializationError{
		BaseError: NewBaseError(AppInitializationErrorCode, message),
	}
}

// AssistantInitializationErrorType represents an assistant initialization error
type AssistantInitializationError struct {
	*BaseError
}

// NewAssistantInitializationError creates a new AssistantInitializationError
func NewAssistantInitializationError(message string) *AssistantInitializationError {
	return &AssistantInitializationError{
		BaseError: NewBaseError(AssistantInitializationErrorCode, message),
	}
}

// AssistantMissingPropertyErrorType represents an assistant missing property error
type AssistantMissingPropertyError struct {
	*BaseError
}

// NewAssistantMissingPropertyError creates a new AssistantMissingPropertyError
func NewAssistantMissingPropertyError(message string) *AssistantMissingPropertyError {
	return &AssistantMissingPropertyError{
		BaseError: NewBaseError(AssistantMissingPropertyErrorCode, message),
	}
}

// AuthorizationErrorType represents an authorization error
type AuthorizationError struct {
	*BaseError
}

// NewAuthorizationError creates a new AuthorizationError
func NewAuthorizationError(message string, original error) *AuthorizationError {
	return &AuthorizationError{
		BaseError: NewBaseErrorWithOriginal(AuthorizationErrorCode, message, original),
	}
}

// ContextMissingPropertyErrorType represents a context missing property error
type ContextMissingPropertyError struct {
	*BaseError
	MissingProperty string
}

// NewContextMissingPropertyError creates a new ContextMissingPropertyError
func NewContextMissingPropertyError(missingProperty, message string) *ContextMissingPropertyError {
	return &ContextMissingPropertyError{
		BaseError:       NewBaseError(ContextMissingPropertyErrorCode, message),
		MissingProperty: missingProperty,
	}
}

// InvalidCustomPropertyErrorType represents an invalid custom property error
type InvalidCustomPropertyError struct {
	*BaseError
}

// NewInvalidCustomPropertyError creates a new InvalidCustomPropertyError
func NewInvalidCustomPropertyError(message string) *InvalidCustomPropertyError {
	return &InvalidCustomPropertyError{
		BaseError: NewBaseError(InvalidCustomPropertyErrorCode, message),
	}
}

// ReceiverMultipleAckErrorType represents a receiver multiple ack error
type ReceiverMultipleAckError struct {
	*BaseError
}

// NewReceiverMultipleAckError creates a new ReceiverMultipleAckError
func NewReceiverMultipleAckError() *ReceiverMultipleAckError {
	return &ReceiverMultipleAckError{
		BaseError: NewBaseError(ReceiverMultipleAckErrorCode, "The receiver's `ack` function was called multiple times."),
	}
}

// ReceiverAuthenticityError represents a receiver authenticity error
type ReceiverAuthenticityError struct {
	*BaseError
}

// NewReceiverAuthenticityError creates a new ReceiverAuthenticityError
func NewReceiverAuthenticityError(message string) *ReceiverAuthenticityError {
	return &ReceiverAuthenticityError{
		BaseError: NewBaseError(ReceiverAuthenticityErrorCode, message),
	}
}

// HTTPReceiverDeferredRequestError represents an HTTP receiver deferred request error
type HTTPReceiverDeferredRequestError struct {
	*BaseError
	Request  *http.Request
	Response http.ResponseWriter
}

// NewHTTPReceiverDeferredRequestError creates a new HTTPReceiverDeferredRequestError
func NewHTTPReceiverDeferredRequestError(message string, req *http.Request, res http.ResponseWriter) *HTTPReceiverDeferredRequestError {
	return &HTTPReceiverDeferredRequestError{
		BaseError: NewBaseError(HTTPReceiverDeferredRequestErrorCode, message),
		Request:   req,
		Response:  res,
	}
}

// MultipleListenerError represents multiple listener errors
type MultipleListenerError struct {
	*BaseError
}

// NewMultipleListenerError creates a new MultipleListenerError
func NewMultipleListenerError(originals []error) *MultipleListenerError {
	message := fmt.Sprintf("Multiple errors occurred while handling several listeners. %d errors occurred.", len(originals))
	return &MultipleListenerError{
		BaseError: &BaseError{
			code:      MultipleListenerErrorCode,
			message:   message,
			originals: originals,
		},
	}
}

// WorkflowStepInitializationError represents a workflow step initialization error
// Deprecated: Workflow steps from apps are no longer supported
type WorkflowStepInitializationError struct {
	*BaseError
}

// NewWorkflowStepInitializationError creates a new WorkflowStepInitializationError
// Deprecated: Workflow steps from apps are no longer supported
func NewWorkflowStepInitializationError(message string) *WorkflowStepInitializationError {
	return &WorkflowStepInitializationError{
		BaseError: NewBaseError(WorkflowStepInitializationErrorCode, message),
	}
}

// CustomFunctionCompleteSuccessError represents a custom function complete success error
type CustomFunctionCompleteSuccessError struct {
	*BaseError
}

// NewCustomFunctionCompleteSuccessError creates a new CustomFunctionCompleteSuccessError
func NewCustomFunctionCompleteSuccessError(message string) *CustomFunctionCompleteSuccessError {
	return &CustomFunctionCompleteSuccessError{
		BaseError: NewBaseError(CustomFunctionCompleteSuccessErrorCode, message),
	}
}

// CustomFunctionCompleteFailError represents a custom function complete fail error
type CustomFunctionCompleteFailError struct {
	*BaseError
}

// NewCustomFunctionCompleteFailError creates a new CustomFunctionCompleteFailError
func NewCustomFunctionCompleteFailError(message string) *CustomFunctionCompleteFailError {
	return &CustomFunctionCompleteFailError{
		BaseError: NewBaseError(CustomFunctionCompleteFailErrorCode, message),
	}
}

// CustomFunctionInitializationError represents a custom function initialization error
type CustomFunctionInitializationError struct {
	*BaseError
}

// NewCustomFunctionInitializationError creates a new CustomFunctionInitializationError
func NewCustomFunctionInitializationError(message string) *CustomFunctionInitializationError {
	return &CustomFunctionInitializationError{
		BaseError: NewBaseError(CustomFunctionInitializationErrorCode, message),
	}
}
