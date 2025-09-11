package types

import (
	"log/slog"

	"github.com/slack-go/slack"
)

// StringIndexed represents a map with string keys and any values
type StringIndexed map[string]interface{}

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
	FunctionInputs interface{} `json:"function_inputs,omitempty"`
	// Retry count of an Events API request
	RetryNum *int `json:"retry_num,omitempty"`
	// Retry reason of an Events API request
	RetryReason *string `json:"retry_reason,omitempty"`

	// Conversation context fields
	Conversation       interface{} `json:"conversation,omitempty"`
	UpdateConversation interface{} `json:"update_conversation,omitempty"`

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
	Channel     *string            `json:"channel,omitempty"`
	Text        *string            `json:"text,omitempty"`
	Blocks      []slack.Block      `json:"blocks,omitempty"`
	Attachments []slack.Attachment `json:"attachments,omitempty"`
	ThreadTS    *string            `json:"thread_ts,omitempty"`
	// Add other ChatPostMessageArguments fields as needed
}

// SayFn represents a function to send a message
type SayFn func(message interface{}) (interface{}, error)

// RespondArguments represents arguments for the respond function
type RespondArguments struct {
	ResponseType    *string            `json:"response_type,omitempty"` // "in_channel" or "ephemeral"
	ReplaceOriginal *bool              `json:"replace_original,omitempty"`
	DeleteOriginal  *bool              `json:"delete_original,omitempty"`
	Text            *string            `json:"text,omitempty"`
	Blocks          []slack.Block      `json:"blocks,omitempty"`
	Attachments     []slack.Attachment `json:"attachments,omitempty"`
}

// RespondFn represents a function to respond to an interaction
type RespondFn func(message interface{}) error

// AckFn represents a function to acknowledge a request
type AckFn[Response any] func(response *Response) error

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
