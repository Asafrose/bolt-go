package types

import (
	"regexp"

	"github.com/slack-go/slack"
)

// SlashCommand is an alias for the slack SDK's SlashCommand
// This provides built-in parsing, validation, and enterprise install support
type SlashCommand = slack.SlashCommand

// CommandConstraints represents constraints for matching commands
type CommandConstraints struct {
	Command string `json:"command,omitempty"`
	// RegExp support
	CommandPattern *regexp.Regexp `json:"-"`
}

// SlackCommandMiddlewareArgs represents arguments for command middleware
type SlackCommandMiddlewareArgs struct {
	AllMiddlewareArgs
	Command SlashCommand           `json:"command"`
	Body    SlashCommand           `json:"body"`    // Strongly typed body
	Payload SlashCommand           `json:"payload"` // Strongly typed payload
	Respond RespondFn              `json:"-"`
	Ack     AckFn[CommandResponse] `json:"-"`
	Say     SayFn                  `json:"-"`
}

// CommandResponse represents a response to a slash command
type CommandResponse struct {
	Text         string             `json:"text,omitempty"`
	ResponseType ResponseType       `json:"response_type,omitempty"` // ResponseTypeInChannel or ResponseTypeEphemeral
	Blocks       []slack.Block      `json:"blocks,omitempty"`
	Attachments  []slack.Attachment `json:"attachments,omitempty"`
}
