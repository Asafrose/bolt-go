package types

import (
	"regexp"

	"github.com/slack-go/slack"
)

// SlackAction represents all known actions from Slack's Block Kit interactive components
type SlackAction interface {
	GetType() string
}

// BlockAction represents a block action
type BlockAction struct {
	Type     string                 `json:"type"`
	BlockID  string                 `json:"block_id"`
	ActionID string                 `json:"action_id"`
	Value    string                 `json:"value,omitempty"`
	Text     *slack.TextBlockObject `json:"text,omitempty"`
}

func (ba BlockAction) GetType() string {
	return ba.Type
}

// InteractiveMessage represents an interactive message action
type InteractiveMessage struct {
	Type       string        `json:"type"`
	CallbackID string        `json:"callback_id"`
	Actions    []interface{} `json:"actions"`
}

func (im InteractiveMessage) GetType() string {
	return im.Type
}

// DialogSubmitAction represents a dialog submission
type DialogSubmitAction struct {
	Type       string                 `json:"type"`
	CallbackID string                 `json:"callback_id"`
	Submission map[string]interface{} `json:"submission"`
}

func (dsa DialogSubmitAction) GetType() string {
	return dsa.Type
}

// WorkflowStepEdit represents a workflow step edit action
type WorkflowStepEdit struct {
	Type               string `json:"type"`
	CallbackID         string `json:"callback_id"`
	WorkflowStepEditID string `json:"workflow_step_edit_id"`
}

func (wse WorkflowStepEdit) GetType() string {
	return wse.Type
}

// ActionConstraints represents constraints for matching actions
type ActionConstraints struct {
	Type       *string `json:"type,omitempty"`
	BlockID    *string `json:"block_id,omitempty"`
	ActionID   *string `json:"action_id,omitempty"`
	CallbackID *string `json:"callback_id,omitempty"`
	// RegExp support
	BlockIDPattern    *regexp.Regexp `json:"-"`
	ActionIDPattern   *regexp.Regexp `json:"-"`
	CallbackIDPattern *regexp.Regexp `json:"-"`
}

// SlackActionMiddlewareArgs represents arguments for action middleware
type SlackActionMiddlewareArgs struct {
	AllMiddlewareArgs
	Payload interface{}        `json:"payload"`
	Action  interface{}        `json:"action"`
	Body    interface{}        `json:"body"`
	Respond RespondFn          `json:"-"`
	Ack     AckFn[interface{}] `json:"-"`
	Say     *SayFn             `json:"-"` // Optional, only for actions with channel context
}

// DialogValidation represents validation errors for dialog submissions
type DialogValidation struct {
	Errors []DialogFieldError `json:"errors"`
}

// DialogFieldError represents a validation error for a dialog field
type DialogFieldError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}
