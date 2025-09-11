package types

import "regexp"

// GlobalShortcut represents a global shortcut
type GlobalShortcut struct {
	Type        string `json:"type"`
	Token       string `json:"token"`
	ActionTS    string `json:"action_ts"`
	TeamID      string `json:"team_id"`
	UserID      string `json:"user_id"`
	CallbackID  string `json:"callback_id"`
	TriggerID   string `json:"trigger_id"`
	ResponseURL string `json:"response_url"`
}

// MessageShortcut represents a message shortcut
type MessageShortcut struct {
	Type        string      `json:"type"`
	Token       string      `json:"token"`
	ActionTS    string      `json:"action_ts"`
	TeamID      string      `json:"team_id"`
	UserID      string      `json:"user_id"`
	CallbackID  string      `json:"callback_id"`
	TriggerID   string      `json:"trigger_id"`
	ResponseURL string      `json:"response_url"`
	MessageTS   string      `json:"message_ts"`
	ChannelID   string      `json:"channel_id"`
	Message     interface{} `json:"message"`
}

// SlackShortcut represents either a global or message shortcut
type SlackShortcut interface {
	GetType() string
	GetCallbackID() string
}

func (gs GlobalShortcut) GetType() string {
	return gs.Type
}

func (gs GlobalShortcut) GetCallbackID() string {
	return gs.CallbackID
}

func (ms MessageShortcut) GetType() string {
	return ms.Type
}

func (ms MessageShortcut) GetCallbackID() string {
	return ms.CallbackID
}

// ShortcutConstraints represents constraints for matching shortcuts
type ShortcutConstraints struct {
	Type       *string `json:"type,omitempty"`
	CallbackID *string `json:"callback_id,omitempty"`
	// RegExp support
	CallbackIDPattern *regexp.Regexp `json:"-"`
}

// SlackShortcutMiddlewareArgs represents arguments for shortcut middleware
type SlackShortcutMiddlewareArgs struct {
	AllMiddlewareArgs
	Shortcut interface{}        `json:"shortcut"`
	Body     interface{}        `json:"body"`
	Payload  interface{}        `json:"payload"`
	Ack      AckFn[interface{}] `json:"-"`
	Say      *SayFn             `json:"-"` // Optional, only for message shortcuts
}
