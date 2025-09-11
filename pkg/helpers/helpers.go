package helpers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Asafrose/bolt-go/pkg/types"
)

// IncomingEventType represents the type of incoming event
type IncomingEventType int

const (
	IncomingEventTypeEvent IncomingEventType = iota
	IncomingEventTypeAction
	IncomingEventTypeCommand
	IncomingEventTypeOptions
	IncomingEventTypeViewAction
	IncomingEventTypeShortcut
)

// EventTypeAndConversation holds event type and conversation info
type EventTypeAndConversation struct {
	Type           *IncomingEventType `json:"type,omitempty"`
	ConversationID *string            `json:"conversation_id,omitempty"`
}

// ParseRequestBody attempts to parse body as JSON first, then as form data
func ParseRequestBody(body []byte) map[string]interface{} {
	var parsed map[string]interface{}

	// Try JSON first
	if err := json.Unmarshal(body, &parsed); err == nil {
		return parsed
	}

	// Try form data
	if values, err := url.ParseQuery(string(body)); err == nil {
		parsed = make(map[string]interface{})
		for key, valueSlice := range values {
			if len(valueSlice) == 1 {
				parsed[key] = valueSlice[0]
			} else if len(valueSlice) > 1 {
				parsed[key] = valueSlice
			}
		}
		return parsed
	}

	return make(map[string]interface{})
}

// GetTypeAndConversation determines the type and conversation ID from a request body
func GetTypeAndConversation(body []byte) EventTypeAndConversation {
	parsed := ParseRequestBody(body)

	// Check for event
	if event, exists := parsed["event"]; exists {
		eventType := IncomingEventTypeEvent
		result := EventTypeAndConversation{Type: &eventType}

		if eventMap, ok := event.(map[string]interface{}); ok {
			// Look for conversation ID in various places
			if channel, exists := eventMap["channel"]; exists {
				if channelStr, ok := channel.(string); ok {
					result.ConversationID = &channelStr
				} else if channelMap, ok := channel.(map[string]interface{}); ok {
					if id, exists := channelMap["id"]; exists {
						if idStr, ok := id.(string); ok {
							result.ConversationID = &idStr
						}
					}
				}
			}

			if channelID, exists := eventMap["channel_id"]; exists {
				if channelIDStr, ok := channelID.(string); ok {
					result.ConversationID = &channelIDStr
				}
			}

			if item, exists := eventMap["item"]; exists {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if channel, exists := itemMap["channel"]; exists {
						if channelStr, ok := channel.(string); ok {
							result.ConversationID = &channelStr
						}
					}
				}
			}
		}

		return result
	}

	// Check for command
	if _, exists := parsed["command"]; exists {
		eventType := IncomingEventTypeCommand
		result := EventTypeAndConversation{Type: &eventType}

		if channelID, exists := parsed["channel_id"]; exists {
			if channelIDStr, ok := channelID.(string); ok {
				result.ConversationID = &channelIDStr
			}
		}

		return result
	}

	// Check for options
	if _, exists := parsed["name"]; exists {
		eventType := IncomingEventTypeOptions
		result := EventTypeAndConversation{Type: &eventType}

		if channel, exists := parsed["channel"]; exists {
			if channelMap, ok := channel.(map[string]interface{}); ok {
				if id, exists := channelMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						result.ConversationID = &idStr
					}
				}
			}
		}

		return result
	}

	if eventTypeStr, exists := parsed["type"]; exists {
		if typeStr, ok := eventTypeStr.(string); ok && typeStr == "block_suggestion" {
			eventType := IncomingEventTypeOptions
			result := EventTypeAndConversation{Type: &eventType}

			// Extract conversation ID from channel
			if channel, exists := parsed["channel"]; exists {
				if channelMap, ok := channel.(map[string]interface{}); ok {
					if id, exists := channelMap["id"]; exists {
						if idStr, ok := id.(string); ok {
							result.ConversationID = &idStr
						}
					}
				}
			}

			return result
		}
	}

	// Check for actions
	if _, exists := parsed["actions"]; exists {
		eventType := IncomingEventTypeAction
		result := EventTypeAndConversation{Type: &eventType}

		if channel, exists := parsed["channel"]; exists {
			if channelMap, ok := channel.(map[string]interface{}); ok {
				if id, exists := channelMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						result.ConversationID = &idStr
					}
				}
			}
		}

		return result
	}

	// Check for dialog submission or workflow step edit
	if eventTypeStr, exists := parsed["type"]; exists {
		if typeStr, ok := eventTypeStr.(string); ok {
			if typeStr == "dialog_submission" || typeStr == "workflow_step_edit" {
				eventType := IncomingEventTypeAction
				return EventTypeAndConversation{Type: &eventType}
			}
			if typeStr == "block_actions" {
				eventType := IncomingEventTypeAction
				result := EventTypeAndConversation{Type: &eventType}

				// Extract conversation ID from channel
				if channel, exists := parsed["channel"]; exists {
					if channelMap, ok := channel.(map[string]interface{}); ok {
						if id, exists := channelMap["id"]; exists {
							if idStr, ok := id.(string); ok {
								result.ConversationID = &idStr
							}
						}
					}
				}

				return result
			}
		}
	}

	// Check for shortcuts
	if eventTypeStr, exists := parsed["type"]; exists {
		if typeStr, ok := eventTypeStr.(string); ok {
			if typeStr == "shortcut" || typeStr == "message_action" {
				eventType := IncomingEventTypeShortcut
				result := EventTypeAndConversation{Type: &eventType}

				if typeStr == "message_action" {
					if channel, exists := parsed["channel"]; exists {
						if channelMap, ok := channel.(map[string]interface{}); ok {
							if id, exists := channelMap["id"]; exists {
								if idStr, ok := id.(string); ok {
									result.ConversationID = &idStr
								}
							}
						}
					}
				}

				return result
			}
		}
	}

	// Check for view submissions/closures
	if view, exists := parsed["view"]; exists {
		if _, ok := view.(map[string]interface{}); ok {
			eventType := IncomingEventTypeViewAction
			return EventTypeAndConversation{Type: &eventType}
		}
	}

	return EventTypeAndConversation{}
}

// IsBodyWithTypeEnterpriseInstall checks if body indicates enterprise install
func IsBodyWithTypeEnterpriseInstall(body []byte) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}

	if isEnterpriseInstall, exists := parsed["is_enterprise_install"]; exists {
		// Handle boolean values
		if isEnterprise, ok := isEnterpriseInstall.(bool); ok {
			return isEnterprise
		}
		// Handle string values (e.g., "true", "false")
		if strValue, ok := isEnterpriseInstall.(string); ok {
			return strValue == "true"
		}
	}

	return false
}

// IsEventTypeToSkipAuthorize checks if event type should skip authorization
func IsEventTypeToSkipAuthorize(eventType string) bool {
	skipTypes := []string{"app_uninstalled", "tokens_revoked"}
	for _, skipType := range skipTypes {
		if eventType == skipType {
			return true
		}
	}
	return false
}

// ExtractEventType extracts the event type from the body
func ExtractEventType(body []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	if event, exists := parsed["event"]; exists {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if eventType, exists := eventMap["type"]; exists {
				if typeStr, ok := eventType.(string); ok {
					return typeStr
				}
			}
		}
	}

	return ""
}

// CreateSayFunction creates a say function for a given channel
func CreateSayFunction(client interface{}, channelID string) types.SayFn {
	return func(message interface{}) (interface{}, error) {
		// Implementation will depend on the actual Slack client
		// For now, return a placeholder
		return nil, nil
	}
}

// CreateRespondFunction creates a respond function for a response URL
func CreateRespondFunction(responseURL string) types.RespondFn {
	return func(message interface{}) error {
		// Implementation will depend on HTTP client for response URL
		// For now, return nil
		return nil
	}
}

// MatchesPattern checks if a string matches a pattern (string or regex)
func MatchesPattern(text string, pattern interface{}) bool {
	switch p := pattern.(type) {
	case string:
		return strings.Contains(text, p)
	case *string:
		if p == nil {
			return true
		}
		return strings.Contains(text, *p)
	case *regexp.Regexp:
		if p == nil {
			return true
		}
		return p.MatchString(text)
	case regexp.Regexp:
		return p.MatchString(text)
	default:
		// For unknown pattern types, return false
		return false
	}
}

// ExtractTeamID extracts team ID from various places in the body
func ExtractTeamID(body []byte) *string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}

	if teamID, exists := parsed["team_id"]; exists {
		if teamIDStr, ok := teamID.(string); ok {
			return &teamIDStr
		}
	}

	if team, exists := parsed["team"]; exists {
		if teamMap, ok := team.(map[string]interface{}); ok {
			if id, exists := teamMap["id"]; exists {
				if idStr, ok := id.(string); ok {
					return &idStr
				}
			}
		}
	}

	return nil
}

// ExtractEnterpriseID extracts enterprise ID from various places in the body
func ExtractEnterpriseID(body []byte) *string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}

	if enterpriseID, exists := parsed["enterprise_id"]; exists {
		if enterpriseIDStr, ok := enterpriseID.(string); ok && enterpriseIDStr != "" {
			return &enterpriseIDStr
		}
	}

	if enterprise, exists := parsed["enterprise"]; exists {
		if enterpriseMap, ok := enterprise.(map[string]interface{}); ok {
			if id, exists := enterpriseMap["id"]; exists {
				if idStr, ok := id.(string); ok && idStr != "" {
					return &idStr
				}
			}
		}
	}

	return nil
}

// ExtractUserID extracts user ID from various places in the body
func ExtractUserID(body []byte) *string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}

	// Check direct user_id field
	if userID, exists := parsed["user_id"]; exists {
		if userIDStr, ok := userID.(string); ok {
			return &userIDStr
		}
	}

	// Check in event
	if event, exists := parsed["event"]; exists {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if userID, exists := eventMap["user"]; exists {
				if userIDStr, ok := userID.(string); ok {
					return &userIDStr
				}
			}
		}
	}

	// Check in user field
	if user, exists := parsed["user"]; exists {
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, exists := userMap["id"]; exists {
				if idStr, ok := id.(string); ok {
					return &idStr
				}
			}
		} else if userStr, ok := user.(string); ok {
			return &userStr
		}
	}

	return nil
}

// VerifySlackSignature verifies the signature of a Slack request
func VerifySlackSignature(signingSecret, signature, timestamp string, body []byte) error {
	if signingSecret == "" {
		return errors.New("signing secret cannot be empty")
	}

	if signature == "" {
		return errors.New("signature cannot be empty")
	}

	if timestamp == "" {
		return errors.New("timestamp cannot be empty")
	}

	// Check signature format
	if !strings.HasPrefix(signature, "v0=") {
		return errors.New("invalid signature format")
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Check if timestamp is too old (more than 5 minutes)
	now := time.Now().Unix()
	if now-ts > 300 {
		return errors.New("timestamp too old")
	}

	// Check if timestamp is in the future (allow 1 minute tolerance)
	if ts-now > 60 {
		return errors.New("timestamp is in the future")
	}

	// Create the signature base string
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

	// Create HMAC-SHA256 hash
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("signature mismatch")
	}

	return nil
}

// IsValidSlackRequest checks if a Slack request is valid
func IsValidSlackRequest(signingSecret, signature, timestamp string, body []byte) bool {
	return VerifySlackSignature(signingSecret, signature, timestamp, body) == nil
}

// GenerateSlackSignature generates a valid Slack signature for testing purposes
func GenerateSlackSignature(signingSecret, baseString string) string {
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
