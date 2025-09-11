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

// EventEnvelope represents the envelope that wraps Slack events from the Events API
type EventEnvelope struct {
	Token              string          `json:"token"`
	TeamID             string          `json:"team_id"`
	APIAppID           string          `json:"api_app_id"`
	Event              SlackEvent      `json:"event"`
	Type               string          `json:"type"` // "event_callback"
	EventID            string          `json:"event_id"`
	EventTime          int64           `json:"event_time"`
	Authorizations     []Authorization `json:"authorizations,omitempty"`
	IsExtSharedChannel bool            `json:"is_ext_shared_channel,omitempty"`
	EventContext       string          `json:"event_context,omitempty"`
}

// Authorization represents an authorization in the event envelope
type Authorization struct {
	EnterpriseID        *string `json:"enterprise_id,omitempty"`
	TeamID              string  `json:"team_id"`
	UserID              string  `json:"user_id"`
	IsBot               bool    `json:"is_bot"`
	IsEnterpriseInstall bool    `json:"is_enterprise_install"`
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
	Event   SlackEvent         `json:"event"` // Strongly typed event
	Body    EventEnvelope      `json:"body"`  // Strongly typed event envelope
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
