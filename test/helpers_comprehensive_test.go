package test

import (
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpersComprehensive implements the missing tests from helpers.spec.ts
func TestHelpersComprehensive(t *testing.T) {
	t.Run("event types", func(t *testing.T) {
		t.Run("should find Event type for generic event", func(t *testing.T) {
			conversationID := "CONVERSATION_ID"
			dummyEventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type":    "app_home_opened",
					"channel": conversationID,
				},
			}

			bodyBytes, err := json.Marshal(dummyEventBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.NotNil(t, typeAndConv.Type, "Type should be detected")
			assert.Equal(t, helpers.IncomingEventTypeEvent, *typeAndConv.Type, "Should detect Event type")
			require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil")
			assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID")
		})
	})

	t.Run("command types", func(t *testing.T) {
		t.Run("should find Command type for generic command", func(t *testing.T) {
			conversationID := "CONVERSATION_ID"
			dummyCommandBody := map[string]interface{}{
				"command":      "COMMAND_NAME",
				"channel_id":   conversationID,
				"response_url": "https://hooks.slack.com/commands/RESPONSE_URL",
			}

			bodyBytes, err := json.Marshal(dummyCommandBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.NotNil(t, typeAndConv.Type, "Type should be detected")
			assert.Equal(t, helpers.IncomingEventTypeCommand, *typeAndConv.Type, "Should detect Command type")
			require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil")
			assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID")
		})
	})

	t.Run("action types", func(t *testing.T) {
		t.Run("should find Action type for block_actions", func(t *testing.T) {
			conversationID := "CONVERSATION_ID"
			dummyActionBody := map[string]interface{}{
				"type": "block_actions",
				"channel": map[string]interface{}{
					"id": conversationID,
				},
				"actions": []interface{}{
					map[string]interface{}{
						"action_id": "test_action",
					},
				},
			}

			bodyBytes, err := json.Marshal(dummyActionBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.NotNil(t, typeAndConv.Type, "Type should be detected for block_actions")
			assert.Equal(t, helpers.IncomingEventTypeAction, *typeAndConv.Type, "Should detect Action type for block_actions")
			require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil for block_actions")
			assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID for block_actions")
		})
	})

	t.Run("shortcut types", func(t *testing.T) {
		shortcutTypes := []string{"shortcut", "message_action"}
		conversationID := "CONVERSATION_ID"

		for _, shortcutType := range shortcutTypes {
			t.Run("should find Shortcut type for "+shortcutType, func(t *testing.T) {
				dummyShortcutBody := map[string]interface{}{
					"type":        shortcutType,
					"callback_id": "test_shortcut",
				}

				// Add conversation ID for message actions
				if shortcutType == "message_action" {
					dummyShortcutBody["channel"] = map[string]interface{}{
						"id": conversationID,
					}
				}

				bodyBytes, err := json.Marshal(dummyShortcutBody)
				require.NoError(t, err)

				typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

				assert.NotNil(t, typeAndConv.Type, "Type should be detected for "+shortcutType)
				assert.Equal(t, helpers.IncomingEventTypeShortcut, *typeAndConv.Type, "Should detect Shortcut type for "+shortcutType)

				// Only message actions have conversation IDs
				if shortcutType == "message_action" {
					require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil for "+shortcutType)
					assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID for "+shortcutType)
				}
			})
		}
	})

	t.Run("view types", func(t *testing.T) {
		viewTypes := []string{"view_submission", "view_closed"}

		for _, viewType := range viewTypes {
			t.Run("should find ViewAction type for "+viewType, func(t *testing.T) {
				dummyViewBody := map[string]interface{}{
					"type": viewType,
					"view": map[string]interface{}{
						"id": "test_view",
					},
				}

				bodyBytes, err := json.Marshal(dummyViewBody)
				require.NoError(t, err)

				typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

				assert.NotNil(t, typeAndConv.Type, "Type should be detected for "+viewType)
				assert.Equal(t, helpers.IncomingEventTypeViewAction, *typeAndConv.Type, "Should detect ViewAction type for "+viewType)
			})
		}
	})

	t.Run("options types", func(t *testing.T) {
		t.Run("should find Options type for block_suggestion", func(t *testing.T) {
			conversationID := "CONVERSATION_ID"
			dummyOptionsBody := map[string]interface{}{
				"type": "block_suggestion",
				"channel": map[string]interface{}{
					"id": conversationID,
				},
				"action_id": "test_action",
			}

			bodyBytes, err := json.Marshal(dummyOptionsBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.NotNil(t, typeAndConv.Type, "Type should be detected for block_suggestion")
			assert.Equal(t, helpers.IncomingEventTypeOptions, *typeAndConv.Type, "Should detect Options type for block_suggestion")
			require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil for block_suggestion")
			assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID for block_suggestion")
		})

		t.Run("should find Options type for name field", func(t *testing.T) {
			conversationID := "CONVERSATION_ID"
			dummyOptionsBody := map[string]interface{}{
				"name": "test_option",
				"channel": map[string]interface{}{
					"id": conversationID,
				},
				"action_id": "test_action",
			}

			bodyBytes, err := json.Marshal(dummyOptionsBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.NotNil(t, typeAndConv.Type, "Type should be detected for options with name field")
			assert.Equal(t, helpers.IncomingEventTypeOptions, *typeAndConv.Type, "Should detect Options type for name field")
			require.NotNil(t, typeAndConv.ConversationID, "Conversation ID should not be nil for name field")
			assert.Equal(t, conversationID, *typeAndConv.ConversationID, "Should extract conversation ID for name field")
		})
	})

	t.Run("invalid events", func(t *testing.T) {
		t.Run("should not find type for invalid event", func(t *testing.T) {
			invalidEventBody := map[string]interface{}{
				"invalid_field": "invalid_value",
			}

			bodyBytes, err := json.Marshal(invalidEventBody)
			require.NoError(t, err)

			typeAndConv := helpers.GetTypeAndConversation(bodyBytes)

			assert.Nil(t, typeAndConv.Type, "Type should not be detected for invalid event")
		})
	})
}

// TestEnterpriseInstallHelpers tests enterprise install detection
func TestEnterpriseInstallHelpers(t *testing.T) {
	t.Run("with body of event type", func(t *testing.T) {
		t.Run("should resolve the is_enterprise_install field", func(t *testing.T) {
			eventBody := map[string]interface{}{
				"is_enterprise_install": true,
				"event": map[string]interface{}{
					"type": "app_home_opened",
				},
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)
			assert.True(t, isEnterpriseInstall, "Should detect enterprise install")
		})

		t.Run("should resolve the is_enterprise_install with provided event type", func(t *testing.T) {
			eventBody := map[string]interface{}{
				"is_enterprise_install": true,
				"event": map[string]interface{}{
					"type": "app_mention",
				},
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)
			assert.True(t, isEnterpriseInstall, "Should detect enterprise install for specific event type")
		})
	})

	t.Run("with is_enterprise_install as a string value", func(t *testing.T) {
		t.Run("should resolve is_enterprise_install as truthy", func(t *testing.T) {
			eventBody := map[string]interface{}{
				"is_enterprise_install": "true", // string value
				"event": map[string]interface{}{
					"type": "app_home_opened",
				},
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)
			assert.True(t, isEnterpriseInstall, "Should detect enterprise install from string value")
		})
	})

	t.Run("with is_enterprise_install as boolean value", func(t *testing.T) {
		t.Run("should resolve is_enterprise_install as truthy", func(t *testing.T) {
			eventBody := map[string]interface{}{
				"is_enterprise_install": true, // boolean value
				"event": map[string]interface{}{
					"type": "app_home_opened",
				},
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)
			assert.True(t, isEnterpriseInstall, "Should detect enterprise install from boolean value")
		})
	})

	t.Run("with is_enterprise_install undefined", func(t *testing.T) {
		t.Run("should resolve is_enterprise_install as falsy", func(t *testing.T) {
			eventBody := map[string]interface{}{
				"event": map[string]interface{}{
					"type": "app_home_opened",
				},
				// is_enterprise_install is not defined
			}

			bodyBytes, err := json.Marshal(eventBody)
			require.NoError(t, err)

			isEnterpriseInstall := helpers.IsBodyWithTypeEnterpriseInstall(bodyBytes)
			assert.False(t, isEnterpriseInstall, "Should not detect enterprise install when undefined")
		})
	})
}

// TestEventTypeSkipHelpers tests event type authorization skipping
func TestEventTypeSkipHelpers(t *testing.T) {
	t.Run("receiver events that can be skipped", func(t *testing.T) {
		t.Run("should return truthy when event can be skipped", func(t *testing.T) {
			skippableEvents := []string{
				"app_uninstalled",
				"tokens_revoked",
			}

			for _, eventType := range skippableEvents {
				canSkip := helpers.IsEventTypeToSkipAuthorize(eventType)
				assert.True(t, canSkip, "Should be able to skip authorization for "+eventType)
			}
		})

		t.Run("should return falsy when event can not be skipped", func(t *testing.T) {
			nonSkippableEvents := []string{
				"app_mention",
				"message",
				"app_home_opened",
			}

			for _, eventType := range nonSkippableEvents {
				canSkip := helpers.IsEventTypeToSkipAuthorize(eventType)
				assert.False(t, canSkip, "Should not be able to skip authorization for "+eventType)
			}
		})

		t.Run("should return falsy when event is invalid", func(t *testing.T) {
			invalidEvents := []string{
				"",
				"invalid_event_type",
				"non_existent_event",
			}

			for _, eventType := range invalidEvents {
				canSkip := helpers.IsEventTypeToSkipAuthorize(eventType)
				assert.False(t, canSkip, "Should not be able to skip authorization for invalid event: "+eventType)
			}
		})
	})
}

// TestEventTypeExtraction tests event type extraction
func TestEventTypeExtraction(t *testing.T) {
	t.Run("should extract event type from event body", func(t *testing.T) {
		eventBody := map[string]interface{}{
			"event": map[string]interface{}{
				"type": "app_mention",
				"text": "Hello bot",
			},
		}

		bodyBytes, err := json.Marshal(eventBody)
		require.NoError(t, err)

		eventType := helpers.ExtractEventType(bodyBytes)
		assert.Equal(t, "app_mention", eventType, "Should extract correct event type")
	})

	t.Run("should return empty string for non-event body", func(t *testing.T) {
		actionBody := map[string]interface{}{
			"type": "block_actions",
			"actions": []interface{}{
				map[string]interface{}{
					"action_id": "test_action",
				},
			},
		}

		bodyBytes, err := json.Marshal(actionBody)
		require.NoError(t, err)

		eventType := helpers.ExtractEventType(bodyBytes)
		assert.Equal(t, "", eventType, "Should return empty string for non-event body")
	})

	t.Run("should return empty string for invalid body", func(t *testing.T) {
		invalidBody := []byte("invalid json")

		eventType := helpers.ExtractEventType(invalidBody)
		assert.Equal(t, "", eventType, "Should return empty string for invalid JSON")
	})
}
