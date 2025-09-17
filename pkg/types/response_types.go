package types

// ResponseType represents valid response types for Slack interactions
// These constants correspond to the response_type values used in Slack's API
// Reference: https://api.slack.com/interactivity/handling#message_responses
type ResponseType string

const (
	// ResponseTypeInChannel makes the response visible to all users in the channel
	ResponseTypeInChannel ResponseType = "in_channel"

	// ResponseTypeEphemeral makes the response visible only to the user who triggered the interaction
	ResponseTypeEphemeral ResponseType = "ephemeral"
)

// String returns the string representation of the response type
func (r ResponseType) String() string {
	return string(r)
}

// IsValid checks if the response type is a valid Slack response type
func (r ResponseType) IsValid() bool {
	switch r {
	case ResponseTypeInChannel, ResponseTypeEphemeral:
		return true
	default:
		return false
	}
}

// AllResponseTypes returns a slice of all valid response types
func AllResponseTypes() []ResponseType {
	return []ResponseType{
		ResponseTypeInChannel,
		ResponseTypeEphemeral,
	}
}
