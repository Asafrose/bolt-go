package helpers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
)

// ParseSlashCommand converts raw JSON data to a strongly typed SlashCommand
func ParseSlashCommand(data map[string]interface{}) (types.SlashCommand, error) {
	command := types.SlashCommand{}

	if cmd, ok := data["command"].(string); ok {
		command.Command = cmd
	}
	if text, ok := data["text"].(string); ok {
		command.Text = text
	}
	if userID, ok := data["user_id"].(string); ok {
		command.UserID = userID
	}
	if userName, ok := data["user_name"].(string); ok {
		command.UserName = userName
	}
	if channelID, ok := data["channel_id"].(string); ok {
		command.ChannelID = channelID
	}
	if channelName, ok := data["channel_name"].(string); ok {
		command.ChannelName = channelName
	}
	if teamID, ok := data["team_id"].(string); ok {
		command.TeamID = teamID
	}
	if teamDomain, ok := data["team_domain"].(string); ok {
		command.TeamDomain = teamDomain
	}
	if responseURL, ok := data["response_url"].(string); ok {
		command.ResponseURL = responseURL
	}
	if triggerID, ok := data["trigger_id"].(string); ok {
		command.TriggerID = triggerID
	}
	if token, ok := data["token"].(string); ok {
		command.Token = token
	}
	if apiAppID, ok := data["api_app_id"].(string); ok {
		command.APIAppID = apiAppID
	}
	if enterpriseID, ok := data["enterprise_id"].(string); ok {
		command.EnterpriseID = enterpriseID
	}
	if enterpriseName, ok := data["enterprise_name"].(string); ok {
		command.EnterpriseName = enterpriseName
	}
	if isEnterpriseInstall, ok := data["is_enterprise_install"].(string); ok {
		// Convert string to bool - Slack sends "true" as string
		command.IsEnterpriseInstall = isEnterpriseInstall == "true"
	} else if isEnterpriseInstall, ok := data["is_enterprise_install"].(bool); ok {
		command.IsEnterpriseInstall = isEnterpriseInstall
	}

	return command, nil
}

// ParseSlackAction converts raw JSON data to a strongly typed SlackAction
func ParseSlackAction(data interface{}) (types.SlackAction, error) {
	// Convert to JSON and back to properly parse the action
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action data: %w", err)
	}

	// Try to determine the action type
	var actionType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(jsonBytes, &actionType); err != nil {
		return nil, fmt.Errorf("failed to determine action type: %w", err)
	}

	// Check if this is a function-scoped action first
	if dataMap, ok := data.(map[string]interface{}); ok {
		if _, hasFunctionExecutionID := dataMap["function_execution_id"]; hasFunctionExecutionID {
			var functionScopedAction types.FunctionScopedAction
			if err := json.Unmarshal(jsonBytes, &functionScopedAction); err != nil {
				return nil, fmt.Errorf("failed to parse function-scoped action: %w", err)
			}
			return functionScopedAction, nil
		}
	}

	switch actionType.Type {
	case "button", "static_select", "multi_static_select", "external_select",
		"multi_external_select", "users_select", "multi_users_select",
		"conversations_select", "multi_conversations_select", "channels_select",
		"multi_channels_select", "overflow", "datepicker", "timepicker",
		"datetime", "radio_buttons", "checkboxes", "plain_text_input",
		"rich_text_input":
		// This is a block action
		var blockAction types.BlockAction
		if err := json.Unmarshal(jsonBytes, &blockAction); err != nil {
			return nil, fmt.Errorf("failed to parse block action: %w", err)
		}
		return blockAction, nil
	case "interactive_message":
		var interactiveMessage types.InteractiveMessage
		if err := json.Unmarshal(jsonBytes, &interactiveMessage); err != nil {
			return nil, fmt.Errorf("failed to parse interactive message: %w", err)
		}
		return interactiveMessage, nil
	case "dialog_submission":
		var dialogSubmit types.DialogSubmitAction
		if err := json.Unmarshal(jsonBytes, &dialogSubmit); err != nil {
			return nil, fmt.Errorf("failed to parse dialog submission: %w", err)
		}
		return dialogSubmit, nil
	case "workflow_step_edit":
		var workflowStepEdit types.WorkflowStepEdit
		if err := json.Unmarshal(jsonBytes, &workflowStepEdit); err != nil {
			return nil, fmt.Errorf("failed to parse workflow step edit: %w", err)
		}
		return workflowStepEdit, nil
	default:
		// Default to block action for unknown types
		var blockAction types.BlockAction
		if err := json.Unmarshal(jsonBytes, &blockAction); err != nil {
			return nil, fmt.Errorf("failed to parse unknown action type as block action: %w", err)
		}
		return blockAction, nil
	}
}

// ParseSlackEvent converts raw JSON data to a strongly typed SlackEvent
func ParseSlackEvent(data interface{}) (types.SlackEvent, error) {
	// For now, we'll create a generic event wrapper since events are complex
	// In the future, this could be expanded to parse specific event types
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Create a generic event that implements SlackEvent
	event := &GenericSlackEvent{}
	if err := json.Unmarshal(jsonBytes, event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	return event, nil
}

// GenericSlackEvent is a generic implementation of SlackEvent
type GenericSlackEvent struct {
	Type    string                 `json:"type"`
	RawData map[string]interface{} `json:"-"`
}

func (e *GenericSlackEvent) GetType() string {
	return e.Type
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve raw data
func (e *GenericSlackEvent) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map to preserve all data
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.RawData = raw

	// Extract the type
	if eventType, ok := raw["type"].(string); ok {
		e.Type = eventType
	}

	return nil
}

// ParseEventEnvelope converts raw JSON data to a strongly typed EventEnvelope
func ParseEventEnvelope(data map[string]interface{}) (types.EventEnvelope, error) {
	envelope := types.EventEnvelope{}

	if token, ok := data["token"].(string); ok {
		envelope.Token = token
	}
	if teamID, ok := data["team_id"].(string); ok {
		envelope.TeamID = teamID
	}
	if apiAppID, ok := data["api_app_id"].(string); ok {
		envelope.APIAppID = apiAppID
	}
	if eventType, ok := data["type"].(string); ok {
		envelope.Type = eventType
	}
	if eventID, ok := data["event_id"].(string); ok {
		envelope.EventID = eventID
	}
	if eventTime, ok := data["event_time"].(float64); ok {
		envelope.EventTime = int64(eventTime)
	}

	// Parse the inner event
	if eventData, exists := data["event"]; exists {
		event, err := ParseSlackEvent(eventData)
		if err != nil {
			return envelope, fmt.Errorf("failed to parse inner event: %w", err)
		}
		envelope.Event = event
	}

	// Parse authorizations if present
	if authData, exists := data["authorizations"]; exists {
		if authList, ok := authData.([]interface{}); ok {
			for _, authItem := range authList {
				if authMap, ok := authItem.(map[string]interface{}); ok {
					auth := types.Authorization{}
					if enterpriseID, ok := authMap["enterprise_id"].(string); ok {
						auth.EnterpriseID = &enterpriseID
					}
					if teamID, ok := authMap["team_id"].(string); ok {
						auth.TeamID = teamID
					}
					if userID, ok := authMap["user_id"].(string); ok {
						auth.UserID = userID
					}
					if isBot, ok := authMap["is_bot"].(bool); ok {
						auth.IsBot = isBot
					}
					if isEnterpriseInstall, ok := authMap["is_enterprise_install"].(bool); ok {
						auth.IsEnterpriseInstall = isEnterpriseInstall
					}
					envelope.Authorizations = append(envelope.Authorizations, auth)
				}
			}
		}
	}

	if isExtSharedChannel, ok := data["is_ext_shared_channel"].(bool); ok {
		envelope.IsExtSharedChannel = isExtSharedChannel
	}
	if eventContext, ok := data["event_context"].(string); ok {
		envelope.EventContext = eventContext
	}

	return envelope, nil
}

// ParseSlackShortcut converts raw JSON data to a strongly typed SlackShortcut
func ParseSlackShortcut(data map[string]interface{}) (types.SlackShortcut, error) {
	// Determine shortcut type
	if shortcutType, exists := data["type"]; exists {
		if typeStr, ok := shortcutType.(string); ok {
			if typeStr == "shortcut" {
				var globalShortcut types.GlobalShortcut
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal global shortcut data: %w", err)
				}
				if err := json.Unmarshal(jsonBytes, &globalShortcut); err != nil {
					return nil, fmt.Errorf("failed to parse global shortcut: %w", err)
				}
				return globalShortcut, nil
			} else if typeStr == "message_action" {
				var messageShortcut types.MessageShortcut
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal message shortcut data: %w", err)
				}
				if err := json.Unmarshal(jsonBytes, &messageShortcut); err != nil {
					return nil, fmt.Errorf("failed to parse message shortcut: %w", err)
				}
				return messageShortcut, nil
			}
		}
	}

	return nil, errors.New("unknown shortcut type")
}

// ParseSlackView converts raw JSON data to a strongly typed SlackView
func ParseSlackView(data map[string]interface{}) (types.SlackView, error) {
	// Determine view type
	if viewType, exists := data["type"]; exists {
		if typeStr, ok := viewType.(string); ok {
			if typeStr == "view_submission" {
				var viewSubmission types.ViewSubmission
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal view submission data: %w", err)
				}
				if err := json.Unmarshal(jsonBytes, &viewSubmission); err != nil {
					return nil, fmt.Errorf("failed to parse view submission: %w", err)
				}
				return viewSubmission, nil
			} else if typeStr == "view_closed" {
				var viewClosed types.ViewClosed
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal view closed data: %w", err)
				}
				if err := json.Unmarshal(jsonBytes, &viewClosed); err != nil {
					return nil, fmt.Errorf("failed to parse view closed: %w", err)
				}
				return viewClosed, nil
			}
		}
	}

	return nil, errors.New("unknown view type")
}

// ParseViewOutput converts raw JSON view data to ViewOutput for processed view data
func ParseViewOutput(data interface{}) (types.ViewOutput, error) {
	output := types.ViewOutput{
		Values: make(map[string]map[string]interface{}),
	}

	// If data is a map containing view information
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Check if there's a "state" field with values
		if state, exists := dataMap["state"]; exists {
			if stateMap, ok := state.(map[string]interface{}); ok {
				if values, exists := stateMap["values"]; exists {
					if valuesMap, ok := values.(map[string]map[string]interface{}); ok {
						output.Values = valuesMap
					} else if valuesInterface, ok := values.(map[string]interface{}); ok {
						// Convert interface{} values to map[string]interface{}
						for k, v := range valuesInterface {
							if vMap, ok := v.(map[string]interface{}); ok {
								if output.Values[k] == nil {
									output.Values[k] = make(map[string]interface{})
								}
								for k2, v2 := range vMap {
									output.Values[k][k2] = v2
								}
							}
						}
					}
				}

				// Try to parse the state as slack.ViewState
				jsonBytes, err := json.Marshal(state)
				if err == nil {
					var viewState slack.ViewState
					if err := json.Unmarshal(jsonBytes, &viewState); err == nil {
						output.State = &viewState
					}
				}
			}
		}

		// Also check for direct "values" field
		if values, exists := dataMap["values"]; exists {
			if valuesMap, ok := values.(map[string]map[string]interface{}); ok {
				output.Values = valuesMap
			}
		}
	}

	return output, nil
}

// ExtractRawDataFromSlackAction extracts raw map data from a strongly typed SlackAction
func ExtractRawDataFromSlackAction(action types.SlackAction) (map[string]interface{}, error) {
	if action == nil {
		return nil, errors.New("action is nil")
	}

	// Marshal and unmarshal to get raw data
	jsonBytes, err := json.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action: %w", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to raw data: %w", err)
	}

	return rawData, nil
}

// ExtractRawDataFromSlackShortcut extracts raw map data from a strongly typed SlackShortcut
func ExtractRawDataFromSlackShortcut(shortcut types.SlackShortcut) (map[string]interface{}, error) {
	if shortcut == nil {
		return nil, errors.New("shortcut is nil")
	}

	// Marshal and unmarshal to get raw data
	jsonBytes, err := json.Marshal(shortcut)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shortcut: %w", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to raw data: %w", err)
	}

	return rawData, nil
}

// ExtractRawDataFromSlackView extracts raw map data from a strongly typed SlackView
func ExtractRawDataFromSlackView(view types.SlackView) (map[string]interface{}, error) {
	if view == nil {
		return nil, errors.New("view is nil")
	}

	// Marshal and unmarshal to get raw data
	jsonBytes, err := json.Marshal(view)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal view: %w", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to raw data: %w", err)
	}

	return rawData, nil
}

// ExtractRawDataFromSlackEvent extracts raw map data from a strongly typed SlackEvent
func ExtractRawDataFromSlackEvent(event types.SlackEvent) (map[string]interface{}, error) {
	if event == nil {
		return nil, errors.New("event is nil")
	}

	// Marshal and unmarshal to get raw data
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to raw data: %w", err)
	}

	return rawData, nil
}
