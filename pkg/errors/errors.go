package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents error codes used throughout the framework
type ErrorCode string

const (
	AppInitializationError ErrorCode = "slack_bolt_app_initialization_error"

	AssistantInitializationError  ErrorCode = "slack_bolt_assistant_initialization_error"
	AssistantMissingPropertyError ErrorCode = "slack_bolt_assistant_missing_property_error"

	AuthorizationError ErrorCode = "slack_bolt_authorization_error"

	ContextMissingPropertyError ErrorCode = "slack_bolt_context_missing_property_error"
	InvalidCustomPropertyError  ErrorCode = "slack_bolt_context_invalid_custom_property_error"

	CustomRouteInitializationError ErrorCode = "slack_bolt_custom_route_initialization_error"

	ReceiverMultipleAckError       ErrorCode = "slack_bolt_receiver_ack_multiple_error"
	ReceiverAuthenticityError      ErrorCode = "slack_bolt_receiver_authenticity_error"
	ReceiverInconsistentStateError ErrorCode = "slack_bolt_receiver_inconsistent_state_error"

	MultipleListenerError ErrorCode = "slack_bolt_multiple_listener_error"

	HTTPReceiverDeferredRequestError ErrorCode = "slack_bolt_http_receiver_deferred_request_error"

	UnknownError ErrorCode = "slack_bolt_unknown_error"

	EventProcessingError ErrorCode = "slack_bolt_event_processing_error"

	WorkflowStepInitializationError ErrorCode = "slack_bolt_workflow_step_initialization_error"

	CustomFunctionInitializationError  ErrorCode = "slack_bolt_custom_function_initialization_error"
	CustomFunctionCompleteSuccessError ErrorCode = "slack_bolt_custom_function_complete_success_error"
	CustomFunctionCompleteFailError    ErrorCode = "slack_bolt_custom_function_complete_fail_error"
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
	_, ok := err.(CodedError)
	return ok
}

// AsCodedError converts an error to a CodedError
func AsCodedError(err error) CodedError {
	if codedErr, ok := err.(CodedError); ok {
		return codedErr
	}
	return NewBaseErrorWithOriginal(UnknownError, err.Error(), err)
}

// Specific error types

// AppInitializationErrorType represents an app initialization error
type AppInitializationErrorType struct {
	*BaseError
}

// NewAppInitializationError creates a new AppInitializationError
func NewAppInitializationError(message string) *AppInitializationErrorType {
	return &AppInitializationErrorType{
		BaseError: NewBaseError(AppInitializationError, message),
	}
}

// AssistantInitializationErrorType represents an assistant initialization error
type AssistantInitializationErrorType struct {
	*BaseError
}

// NewAssistantInitializationError creates a new AssistantInitializationError
func NewAssistantInitializationError(message string) *AssistantInitializationErrorType {
	return &AssistantInitializationErrorType{
		BaseError: NewBaseError(AssistantInitializationError, message),
	}
}

// AssistantMissingPropertyErrorType represents an assistant missing property error
type AssistantMissingPropertyErrorType struct {
	*BaseError
}

// NewAssistantMissingPropertyError creates a new AssistantMissingPropertyError
func NewAssistantMissingPropertyError(message string) *AssistantMissingPropertyErrorType {
	return &AssistantMissingPropertyErrorType{
		BaseError: NewBaseError(AssistantMissingPropertyError, message),
	}
}

// AuthorizationErrorType represents an authorization error
type AuthorizationErrorType struct {
	*BaseError
}

// NewAuthorizationError creates a new AuthorizationError
func NewAuthorizationError(message string, original error) *AuthorizationErrorType {
	return &AuthorizationErrorType{
		BaseError: NewBaseErrorWithOriginal(AuthorizationError, message, original),
	}
}

// ContextMissingPropertyErrorType represents a context missing property error
type ContextMissingPropertyErrorType struct {
	*BaseError
	MissingProperty string
}

// NewContextMissingPropertyError creates a new ContextMissingPropertyError
func NewContextMissingPropertyError(missingProperty, message string) *ContextMissingPropertyErrorType {
	return &ContextMissingPropertyErrorType{
		BaseError:       NewBaseError(ContextMissingPropertyError, message),
		MissingProperty: missingProperty,
	}
}

// InvalidCustomPropertyErrorType represents an invalid custom property error
type InvalidCustomPropertyErrorType struct {
	*BaseError
}

// NewInvalidCustomPropertyError creates a new InvalidCustomPropertyError
func NewInvalidCustomPropertyError(message string) *InvalidCustomPropertyErrorType {
	return &InvalidCustomPropertyErrorType{
		BaseError: NewBaseError(InvalidCustomPropertyError, message),
	}
}

// ReceiverMultipleAckErrorType represents a receiver multiple ack error
type ReceiverMultipleAckErrorType struct {
	*BaseError
}

// NewReceiverMultipleAckError creates a new ReceiverMultipleAckError
func NewReceiverMultipleAckError() *ReceiverMultipleAckErrorType {
	return &ReceiverMultipleAckErrorType{
		BaseError: NewBaseError(ReceiverMultipleAckError, "The receiver's `ack` function was called multiple times."),
	}
}

// ReceiverAuthenticityErrorType represents a receiver authenticity error
type ReceiverAuthenticityErrorType struct {
	*BaseError
}

// NewReceiverAuthenticityError creates a new ReceiverAuthenticityError
func NewReceiverAuthenticityError(message string) *ReceiverAuthenticityErrorType {
	return &ReceiverAuthenticityErrorType{
		BaseError: NewBaseError(ReceiverAuthenticityError, message),
	}
}

// HTTPReceiverDeferredRequestErrorType represents an HTTP receiver deferred request error
type HTTPReceiverDeferredRequestErrorType struct {
	*BaseError
	Request  *http.Request
	Response http.ResponseWriter
}

// NewHTTPReceiverDeferredRequestError creates a new HTTPReceiverDeferredRequestError
func NewHTTPReceiverDeferredRequestError(message string, req *http.Request, res http.ResponseWriter) *HTTPReceiverDeferredRequestErrorType {
	return &HTTPReceiverDeferredRequestErrorType{
		BaseError: NewBaseError(HTTPReceiverDeferredRequestError, message),
		Request:   req,
		Response:  res,
	}
}

// MultipleListenerErrorType represents multiple listener errors
type MultipleListenerErrorType struct {
	*BaseError
}

// NewMultipleListenerError creates a new MultipleListenerError
func NewMultipleListenerError(originals []error) *MultipleListenerErrorType {
	message := fmt.Sprintf("Multiple errors occurred while handling several listeners. %d errors occurred.", len(originals))
	return &MultipleListenerErrorType{
		BaseError: &BaseError{
			code:      MultipleListenerError,
			message:   message,
			originals: originals,
		},
	}
}

// WorkflowStepInitializationErrorType represents a workflow step initialization error
// Deprecated: Workflow steps from apps are no longer supported
type WorkflowStepInitializationErrorType struct {
	*BaseError
}

// NewWorkflowStepInitializationError creates a new WorkflowStepInitializationError
// Deprecated: Workflow steps from apps are no longer supported
func NewWorkflowStepInitializationError(message string) *WorkflowStepInitializationErrorType {
	return &WorkflowStepInitializationErrorType{
		BaseError: NewBaseError(WorkflowStepInitializationError, message),
	}
}

// CustomFunctionCompleteSuccessErrorType represents a custom function complete success error
type CustomFunctionCompleteSuccessErrorType struct {
	*BaseError
}

// NewCustomFunctionCompleteSuccessError creates a new CustomFunctionCompleteSuccessError
func NewCustomFunctionCompleteSuccessError(message string) *CustomFunctionCompleteSuccessErrorType {
	return &CustomFunctionCompleteSuccessErrorType{
		BaseError: NewBaseError(CustomFunctionCompleteSuccessError, message),
	}
}

// CustomFunctionCompleteFailErrorType represents a custom function complete fail error
type CustomFunctionCompleteFailErrorType struct {
	*BaseError
}

// NewCustomFunctionCompleteFailError creates a new CustomFunctionCompleteFailError
func NewCustomFunctionCompleteFailError(message string) *CustomFunctionCompleteFailErrorType {
	return &CustomFunctionCompleteFailErrorType{
		BaseError: NewBaseError(CustomFunctionCompleteFailError, message),
	}
}

// CustomFunctionInitializationErrorType represents a custom function initialization error
type CustomFunctionInitializationErrorType struct {
	*BaseError
}

// NewCustomFunctionInitializationError creates a new CustomFunctionInitializationError
func NewCustomFunctionInitializationError(message string) *CustomFunctionInitializationErrorType {
	return &CustomFunctionInitializationErrorType{
		BaseError: NewBaseError(CustomFunctionInitializationError, message),
	}
}
