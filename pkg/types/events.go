package types

import (
	"regexp"

	"github.com/slack-go/slack/slackevents"
)

// FunctionInputs represents inputs provided to a function when executed
type FunctionInputs map[string]interface{}

// SlackEvent represents a Slack event
type SlackEvent interface {
	GetType() string
}

// EventConstraints represents constraints for matching events
type EventConstraints struct {
	Type    *string `json:"type,omitempty"`
	Subtype *string `json:"subtype,omitempty"`
	// RegExp support
	TypePattern *regexp.Regexp `json:"-"`
}

// SlackEventMiddlewareArgs represents arguments for event middleware
type SlackEventMiddlewareArgs struct {
	AllMiddlewareArgs
	Event   interface{}        `json:"event"`
	Body    interface{}        `json:"body"`
	Message *MessageEvent      `json:"message,omitempty"`
	Say     SayFn              `json:"-"`
	Ack     AckFn[interface{}] `json:"-"`
}

// MessageEvent represents a message event with additional context
type MessageEvent struct {
	slackevents.MessageEvent
	// Additional fields that might be needed
	BotID      *string     `json:"bot_id,omitempty"`
	BotProfile *BotProfile `json:"bot_profile,omitempty"`
}

// BotProfile represents a bot profile
type BotProfile struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	AppID   string            `json:"app_id"`
	Icons   map[string]string `json:"icons"`
	Updated int64             `json:"updated"`
	TeamID  string            `json:"team_id"`
}

// MessageConstraints represents constraints for matching messages
type MessageConstraints struct {
	Pattern       *string `json:"pattern,omitempty"`
	Subtype       *string `json:"subtype,omitempty"`
	DirectMention bool    `json:"direct_mention"`
	// RegExp support
	PatternRegexp *regexp.Regexp `json:"-"`
}
