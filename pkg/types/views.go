package types

import (
	"regexp"

	"github.com/slack-go/slack"
)

// ViewSubmission represents a view submission
type ViewSubmission struct {
	Type      string                 `json:"type"`
	TeamID    string                 `json:"team_id"`
	UserID    string                 `json:"user_id"`
	APIAppID  string                 `json:"api_app_id"`
	Token     string                 `json:"token"`
	TriggerID string                 `json:"trigger_id"`
	View      slack.ModalViewRequest `json:"view"`
	Response  ResponseURLs           `json:"response_urls,omitempty"`
}

// ViewClosed represents a view closed event
type ViewClosed struct {
	Type     string                 `json:"type"`
	TeamID   string                 `json:"team_id"`
	UserID   string                 `json:"user_id"`
	APIAppID string                 `json:"api_app_id"`
	Token    string                 `json:"token"`
	View     slack.ModalViewRequest `json:"view"`
}

// SlackView represents either a view submission or view closed event
type SlackView interface {
	GetType() string
}

func (vs ViewSubmission) GetType() string {
	return vs.Type
}

func (vc ViewClosed) GetType() string {
	return vc.Type
}

// ResponseURLs represents response URLs in view submissions
type ResponseURLs []ResponseURL

// ResponseURL represents a single response URL
type ResponseURL struct {
	ResponseURL string `json:"response_url"`
	ChannelID   string `json:"channel_id"`
}

// ViewConstraints represents constraints for matching views
type ViewConstraints struct {
	Type       string `json:"type,omitempty"`
	CallbackID string `json:"callback_id,omitempty"`
	ViewID     string `json:"view_id,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
	// RegExp support
	CallbackIDPattern *regexp.Regexp `json:"-"`
	ViewIDPattern     *regexp.Regexp `json:"-"`
	ExternalIDPattern *regexp.Regexp `json:"-"`
}

// ViewOutput represents the processed view data
type ViewOutput struct {
	State  *slack.ViewState                  `json:"state,omitempty"`
	Values map[string]map[string]interface{} `json:"values,omitempty"`
}

// SlackViewMiddlewareArgs represents arguments for view middleware
type SlackViewMiddlewareArgs struct {
	AllMiddlewareArgs
	View    ViewOutput          `json:"view"`    // Strongly typed processed view data
	Body    SlackView           `json:"body"`    // Strongly typed view action
	Payload ViewOutput          `json:"payload"` // Strongly typed payload (same as view)
	Ack     AckFn[ViewResponse] `json:"-"`
}

// ViewResponse represents a response to a view submission
type ViewResponse struct {
	ResponseAction string                  `json:"response_action,omitempty"` // "clear", "update", "push", "errors"
	View           *slack.ModalViewRequest `json:"view,omitempty"`
	Errors         map[string]string       `json:"errors,omitempty"`
}
