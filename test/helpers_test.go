package test

import (
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/stretchr/testify/assert"
)

func TestGetTypeAndConversation(t *testing.T) {
	t.Parallel()
	t.Run("should identify event_callback type", func(t *testing.T) {
		body := map[string]interface{}{
			"type":    "event_callback",
			"team_id": "T123456",
			"event": map[string]interface{}{
				"type":    "app_mention",
				"channel": "C123456",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeEvent, *result.Type)
		assert.NotNil(t, result.ConversationID, "Conversation ID should be extracted")
		assert.Equal(t, "C123456", *result.ConversationID)
	})

	t.Run("should identify slash command type", func(t *testing.T) {
		body := map[string]interface{}{
			"token":      "verification-token",
			"team_id":    "T123456",
			"channel_id": "C123456",
			"user_id":    "U123456",
			"command":    "/test",
			"text":       "hello",
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeCommand, *result.Type)
		assert.NotNil(t, result.ConversationID, "Conversation ID should be extracted")
		assert.Equal(t, "C123456", *result.ConversationID)
	})

	t.Run("should identify interactive component type", func(t *testing.T) {
		body := map[string]interface{}{
			"type":    "block_actions",
			"token":   "verification-token",
			"team":    map[string]interface{}{"id": "T123456"},
			"channel": map[string]interface{}{"id": "C123456"},
			"user":    map[string]interface{}{"id": "U123456"},
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "button_1",
					"type":      "button",
				},
			},
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeAction, *result.Type)
		assert.NotNil(t, result.ConversationID, "Conversation ID should be extracted")
		assert.Equal(t, "C123456", *result.ConversationID)
	})

	t.Run("should identify shortcut type", func(t *testing.T) {
		body := map[string]interface{}{
			"type":        "shortcut",
			"token":       "verification-token",
			"team":        map[string]interface{}{"id": "T123456"},
			"user":        map[string]interface{}{"id": "U123456"},
			"callback_id": "test_shortcut",
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeShortcut, *result.Type)
		// Global shortcuts don't have conversation IDs
		assert.Nil(t, result.ConversationID, "Global shortcut should not have conversation ID")
	})

	t.Run("should identify view submission type", func(t *testing.T) {
		body := map[string]interface{}{
			"type":  "view_submission",
			"token": "verification-token",
			"team":  map[string]interface{}{"id": "T123456"},
			"user":  map[string]interface{}{"id": "U123456"},
			"view": map[string]interface{}{
				"id":          "V123456",
				"callback_id": "test_modal",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeViewAction, *result.Type)
		// Views don't typically have conversation IDs
		assert.Nil(t, result.ConversationID, "View submission should not have conversation ID")
	})

	t.Run("should identify options request type", func(t *testing.T) {
		body := map[string]interface{}{
			"type":      "block_suggestion",
			"token":     "verification-token",
			"team":      map[string]interface{}{"id": "T123456"},
			"user":      map[string]interface{}{"id": "U123456"},
			"channel":   map[string]interface{}{"id": "C123456"},
			"action_id": "select_1",
			"value":     "te",
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.NotNil(t, result.Type, "Type should be identified")
		assert.Equal(t, helpers.IncomingEventTypeOptions, *result.Type)
		assert.NotNil(t, result.ConversationID, "Conversation ID should be extracted")
		assert.Equal(t, "C123456", *result.ConversationID)
	})

	t.Run("should handle unknown type", func(t *testing.T) {
		body := map[string]interface{}{
			"unknown_field": "unknown_value",
		}

		bodyBytes, _ := json.Marshal(body)
		result := helpers.GetTypeAndConversation(bodyBytes)

		assert.Nil(t, result.Type, "Unknown type should return nil")
		assert.Nil(t, result.ConversationID, "Unknown type should not have conversation ID")
	})

	t.Run("should handle malformed JSON", func(t *testing.T) {
		malformedJSON := []byte(`{"type": "event_callback", "malformed": }`)
		result := helpers.GetTypeAndConversation(malformedJSON)

		assert.Nil(t, result.Type, "Malformed JSON should return nil type")
		assert.Nil(t, result.ConversationID, "Malformed JSON should not have conversation ID")
	})
}

func TestExtractTeamID(t *testing.T) {
	t.Parallel()
	t.Run("should extract team_id from root level", func(t *testing.T) {
		body := map[string]interface{}{
			"team_id": "T123456",
		}

		bodyBytes, _ := json.Marshal(body)
		teamID := helpers.ExtractTeamID(bodyBytes)

		assert.NotNil(t, teamID, "Team ID should be extracted")
		assert.Equal(t, "T123456", *teamID)
	})

	t.Run("should extract team_id from team object", func(t *testing.T) {
		body := map[string]interface{}{
			"team": map[string]interface{}{
				"id": "T123456",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		teamID := helpers.ExtractTeamID(bodyBytes)

		assert.NotNil(t, teamID, "Team ID should be extracted from team object")
		assert.Equal(t, "T123456", *teamID)
	})

	t.Run("should return nil when team_id not found", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id": "U123456",
		}

		bodyBytes, _ := json.Marshal(body)
		teamID := helpers.ExtractTeamID(bodyBytes)

		assert.Nil(t, teamID, "Should return nil when team_id not found")
	})

	t.Run("should handle malformed JSON", func(t *testing.T) {
		malformedJSON := []byte(`{"team_id": }`)
		teamID := helpers.ExtractTeamID(malformedJSON)

		assert.Nil(t, teamID, "Should return nil for malformed JSON")
	})
}

func TestExtractEnterpriseID(t *testing.T) {
	t.Parallel()
	t.Run("should extract enterprise_id from root level", func(t *testing.T) {
		body := map[string]interface{}{
			"enterprise_id": "E123456",
		}

		bodyBytes, _ := json.Marshal(body)
		enterpriseID := helpers.ExtractEnterpriseID(bodyBytes)

		assert.NotNil(t, enterpriseID, "Enterprise ID should be extracted")
		assert.Equal(t, "E123456", *enterpriseID)
	})

	t.Run("should extract enterprise_id from enterprise object", func(t *testing.T) {
		body := map[string]interface{}{
			"enterprise": map[string]interface{}{
				"id": "E123456",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		enterpriseID := helpers.ExtractEnterpriseID(bodyBytes)

		assert.NotNil(t, enterpriseID, "Enterprise ID should be extracted from enterprise object")
		assert.Equal(t, "E123456", *enterpriseID)
	})

	t.Run("should return nil when enterprise_id not found", func(t *testing.T) {
		body := map[string]interface{}{
			"team_id": "T123456",
		}

		bodyBytes, _ := json.Marshal(body)
		enterpriseID := helpers.ExtractEnterpriseID(bodyBytes)

		assert.Nil(t, enterpriseID, "Should return nil when enterprise_id not found")
	})
}

func TestExtractUserID(t *testing.T) {
	t.Parallel()
	t.Run("should extract user_id from root level", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id": "U123456",
		}

		bodyBytes, _ := json.Marshal(body)
		userID := helpers.ExtractUserID(bodyBytes)

		assert.NotNil(t, userID, "User ID should be extracted")
		assert.Equal(t, "U123456", *userID)
	})

	t.Run("should extract user_id from user object", func(t *testing.T) {
		body := map[string]interface{}{
			"user": map[string]interface{}{
				"id": "U123456",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		userID := helpers.ExtractUserID(bodyBytes)

		assert.NotNil(t, userID, "User ID should be extracted from user object")
		assert.Equal(t, "U123456", *userID)
	})

	t.Run("should extract user_id from event", func(t *testing.T) {
		body := map[string]interface{}{
			"event": map[string]interface{}{
				"user": "U123456",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		userID := helpers.ExtractUserID(bodyBytes)

		assert.NotNil(t, userID, "User ID should be extracted from event")
		assert.Equal(t, "U123456", *userID)
	})

	t.Run("should return nil when user_id not found", func(t *testing.T) {
		body := map[string]interface{}{
			"team_id": "T123456",
		}

		bodyBytes, _ := json.Marshal(body)
		userID := helpers.ExtractUserID(bodyBytes)

		assert.Nil(t, userID, "Should return nil when user_id not found")
	})
}

func TestExtractEventType(t *testing.T) {
	t.Parallel()
	t.Run("should extract event type from event object", func(t *testing.T) {
		body := map[string]interface{}{
			"type": "event_callback",
			"event": map[string]interface{}{
				"type": "app_mention",
			},
		}

		bodyBytes, _ := json.Marshal(body)
		eventType := helpers.ExtractEventType(bodyBytes)

		assert.Equal(t, "app_mention", eventType, "Event type should be extracted")
	})

	t.Run("should return empty string when event type not found", func(t *testing.T) {
		body := map[string]interface{}{
			"type": "event_callback",
		}

		bodyBytes, _ := json.Marshal(body)
		eventType := helpers.ExtractEventType(bodyBytes)

		assert.Equal(t, "", eventType, "Should return empty string when event type not found")
	})

	t.Run("should handle non-event_callback types", func(t *testing.T) {
		body := map[string]interface{}{
			"type": "block_actions",
		}

		bodyBytes, _ := json.Marshal(body)
		eventType := helpers.ExtractEventType(bodyBytes)

		assert.Equal(t, "", eventType, "Should return empty string for non-event types")
	})
}

func TestIsBodyWithTypeEnterpriseInstall(t *testing.T) {
	t.Parallel()
	t.Run("should identify enterprise install", func(t *testing.T) {
		body := map[string]interface{}{
			"is_enterprise_install": true,
		}

		bodyBytes, _ := json.Marshal(body)
		isEnterprise := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)

		assert.True(t, isEnterprise, "Should identify enterprise install")
	})

	t.Run("should identify non-enterprise install", func(t *testing.T) {
		body := map[string]interface{}{
			"is_enterprise_install": false,
		}

		bodyBytes, _ := json.Marshal(body)
		isEnterprise := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)

		assert.False(t, isEnterprise, "Should identify non-enterprise install")
	})

	t.Run("should default to false when field missing", func(t *testing.T) {
		body := map[string]interface{}{
			"team_id": "T123456",
		}

		bodyBytes, _ := json.Marshal(body)
		isEnterprise := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)

		assert.False(t, isEnterprise, "Should default to false when field missing")
	})
}

func TestIsEventTypeToSkipAuthorize(t *testing.T) {
	t.Parallel()
	t.Run("should skip authorization for app_uninstalled", func(t *testing.T) {
		shouldSkip := helpers.IsEventTypeToSkipAuthorize("app_uninstalled")
		assert.True(t, shouldSkip, "Should skip authorization for app_uninstalled")
	})

	t.Run("should skip authorization for tokens_revoked", func(t *testing.T) {
		shouldSkip := helpers.IsEventTypeToSkipAuthorize("tokens_revoked")
		assert.True(t, shouldSkip, "Should skip authorization for tokens_revoked")
	})

	t.Run("should not skip authorization for regular events", func(t *testing.T) {
		shouldSkip := helpers.IsEventTypeToSkipAuthorize("app_mention")
		assert.False(t, shouldSkip, "Should not skip authorization for regular events")
	})

	t.Run("should not skip authorization for message events", func(t *testing.T) {
		shouldSkip := helpers.IsEventTypeToSkipAuthorize("message")
		assert.False(t, shouldSkip, "Should not skip authorization for message events")
	})

	t.Run("should not skip authorization for empty event type", func(t *testing.T) {
		shouldSkip := helpers.IsEventTypeToSkipAuthorize("")
		assert.False(t, shouldSkip, "Should not skip authorization for empty event type")
	})
}

func TestConversationIDExtraction(t *testing.T) {
	t.Parallel()
	t.Run("should extract conversation ID from different event types", func(t *testing.T) {
		testCases := []struct {
			name           string
			body           map[string]interface{}
			expectedConvID *string
		}{
			{
				name: "message event",
				body: map[string]interface{}{
					"type": "event_callback",
					"event": map[string]interface{}{
						"type":    "message",
						"channel": "C123456",
					},
				},
				expectedConvID: stringPtr("C123456"),
			},
			{
				name: "block action with channel",
				body: map[string]interface{}{
					"type":    "block_actions",
					"channel": map[string]interface{}{"id": "C123456"},
				},
				expectedConvID: stringPtr("C123456"),
			},
			{
				name: "slash command",
				body: map[string]interface{}{
					"command":    "/test",
					"channel_id": "C123456",
				},
				expectedConvID: stringPtr("C123456"),
			},
			{
				name: "global shortcut",
				body: map[string]interface{}{
					"type":        "shortcut",
					"callback_id": "global_shortcut",
				},
				expectedConvID: nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				bodyBytes, _ := json.Marshal(tc.body)
				result := helpers.GetTypeAndConversation(bodyBytes)

				if tc.expectedConvID == nil {
					assert.Nil(t, result.ConversationID, "Conversation ID should be nil")
				} else {
					assert.NotNil(t, result.ConversationID, "Conversation ID should not be nil")
					assert.Equal(t, *tc.expectedConvID, *result.ConversationID, "Conversation ID should match")
				}
			})
		}
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
