package types

import (
	"log/slog"
	"time"

	"github.com/slack-go/slack"
)

// StringIndexed represents a map with string keys and any values
type StringIndexed map[string]interface{}

// UpdateConversationFn represents a function to update conversation state
type UpdateConversationFn func(conversation any, expiresAt *time.Time) error

// Context provides contextual information associated with an incoming request
type Context struct {
	// A bot token, which starts with `xoxb-`
	BotToken *string `json:"bot_token,omitempty"`
	// A user token, which starts with `xoxp-`
	UserToken *string `json:"user_token,omitempty"`
	// This app's bot ID in the installed workspace
	BotID *string `json:"bot_id,omitempty"`
	// This app's bot user ID in the installed workspace
	BotUserID *string `json:"bot_user_id,omitempty"`
	// User ID
	UserID *string `json:"user_id,omitempty"`
	// Workspace ID
	TeamID *string `json:"team_id,omitempty"`
	// Enterprise Grid Organization ID
	EnterpriseID *string `json:"enterprise_id,omitempty"`
	// Is the app installed at an Enterprise level?
	IsEnterpriseInstall bool `json:"is_enterprise_install"`
	// A JIT and function-specific token
	FunctionBotAccessToken *string `json:"function_bot_access_token,omitempty"`
	// Function execution ID associated with the event
	FunctionExecutionID *string `json:"function_execution_id,omitempty"`
	// Inputs that were provided to a function when it was executed
	FunctionInputs FunctionInputs `json:"function_inputs,omitempty"`
	// Retry count of an Events API request
	RetryNum *int `json:"retry_num,omitempty"`
	// Retry reason of an Events API request
	RetryReason *string `json:"retry_reason,omitempty"`

	// Conversation context fields
	Conversation       any                  `json:"conversation,omitempty"`
	UpdateConversation UpdateConversationFn `json:"-"` // Function, not serialized

	// Custom properties
	Custom StringIndexed `json:"custom,omitempty"`
}

// NextFn represents the next function in middleware chain
type NextFn func() error

// AllMiddlewareArgs contains common arguments for all middleware
type AllMiddlewareArgs struct {
	Context *Context      `json:"context"`
	Logger  *slog.Logger  `json:"logger"`
	Client  *slack.Client `json:"client"`
	Next    NextFn        `json:"-"`
}

// Middleware represents a middleware function
type Middleware[Args any] func(args Args) error

// SayArguments represents arguments for the say function
type SayArguments struct {
	Channel     *string              `json:"channel,omitempty"`
	Text        *string              `json:"text,omitempty"`
	Blocks      []slack.Block        `json:"blocks,omitempty"`
	Attachments []slack.Attachment   `json:"attachments,omitempty"`
	ThreadTS    *string              `json:"thread_ts,omitempty"`
	Metadata    *slack.SlackMetadata `json:"metadata,omitempty"`
	// Add other ChatPostMessageArguments fields as needed
}

// SayMessage represents the union type for SayFn parameter: string | SayArguments
type SayMessage interface {
	isSayMessage()
}

// String message implementation
type SayString string

func (s SayString) isSayMessage() {}

// SayArguments message implementation
func (s SayArguments) isSayMessage() {}

// SayResponse represents the response from a say operation
type SayResponse struct {
	*slack.Channel
	*slack.Message
	Timestamp string `json:"ts,omitempty"`
}

// SayFn represents a function to send a message
type SayFn func(message SayMessage) (*SayResponse, error)

// RespondArguments represents arguments for the respond function
type RespondArguments struct {
	ResponseType    *string            `json:"response_type,omitempty"` // "in_channel" or "ephemeral"
	ReplaceOriginal *bool              `json:"replace_original,omitempty"`
	DeleteOriginal  *bool              `json:"delete_original,omitempty"`
	Text            *string            `json:"text,omitempty"`
	Blocks          []slack.Block      `json:"blocks,omitempty"`
	Attachments     []slack.Attachment `json:"attachments,omitempty"`
}

// RespondMessage represents the union type for RespondFn parameter: string | RespondArguments
type RespondMessage interface {
	isRespondMessage()
}

// String message implementation
type RespondString string

func (r RespondString) isRespondMessage() {}

// RespondArguments message implementation
func (r RespondArguments) isRespondMessage() {}

// RespondFn represents a function to respond to an interaction
type RespondFn func(message RespondMessage) error

// AckResponse represents union types for AckFn responses
type AckResponse interface {
	isAckResponse()
}

// Common AckFn response types
type AckVoid struct{}
type AckString string

func (a AckVoid) isAckResponse()          {}
func (a AckString) isAckResponse()        {}
func (s SayArguments) isAckResponse()     {} // For commands: string | SayArguments
func (r RespondArguments) isAckResponse() {} // For commands: string | RespondArguments

// AckFn represents a function to acknowledge a request
type AckFn[Response any] func(response *Response) error

// Specific AckFn types for common use cases
type AckVoidFn func() error
type AckStringFn func(response *string) error
type AckStringOrSayArgsFn func(response AckResponse) error     // string | SayArguments
type AckStringOrRespondArgsFn func(response AckResponse) error // string | RespondArguments

// BuiltinContextKeys lists the built-in context keys
var BuiltinContextKeys = []string{
	"bot_token",
	"user_token",
	"bot_id",
	"bot_user_id",
	"team_id",
	"enterprise_id",
	"function_bot_access_token",
	"function_execution_id",
	"function_inputs",
	"retry_num",
	"retry_reason",
}
