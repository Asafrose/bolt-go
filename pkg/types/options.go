package types

import (
	"regexp"

	"github.com/slack-go/slack"
)

// OptionsRequest represents an options request
type OptionsRequest struct {
	Type      string      `json:"type"`
	Token     string      `json:"token"`
	TeamID    string      `json:"team_id"`
	UserID    string      `json:"user_id"`
	APIAppID  string      `json:"api_app_id"`
	BlockID   string      `json:"block_id,omitempty"`
	ActionID  string      `json:"action_id,omitempty"`
	Value     string      `json:"value,omitempty"`
	Container interface{} `json:"container,omitempty"`
}

// OptionsConstraints represents constraints for matching options requests
type OptionsConstraints struct {
	BlockID  string `json:"block_id,omitempty"`
	ActionID string `json:"action_id,omitempty"`
	// RegExp support
	BlockIDPattern  *regexp.Regexp `json:"-"`
	ActionIDPattern *regexp.Regexp `json:"-"`
}

// SlackOptionsMiddlewareArgs represents arguments for options middleware
type SlackOptionsMiddlewareArgs struct {
	AllMiddlewareArgs
	Options OptionsRequest         `json:"options"`
	Body    interface{}            `json:"body"`
	Payload interface{}            `json:"payload"`
	Ack     AckFn[OptionsResponse] `json:"-"`
}

// OptionsResponse represents a response to an options request
type OptionsResponse struct {
	Options      []Option      `json:"options,omitempty"`
	OptionGroups []OptionGroup `json:"option_groups,omitempty"`
}

// Option is an alias for the slack SDK's OptionBlockObject
// This provides built-in validation and proper JSON marshaling
type Option = slack.OptionBlockObject

// OptionGroup is an alias for the slack SDK's OptionGroupBlockObject
// This provides built-in validation and proper JSON marshaling
type OptionGroup = slack.OptionGroupBlockObject

// TextObject is an alias for the slack SDK's TextBlockObject
// This provides built-in validation and proper JSON marshaling
type TextObject = slack.TextBlockObject
