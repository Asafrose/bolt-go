package test

import (
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/types"
)

// Test constants
var (
	fakeToken         = "xoxb-fake-token" //nolint:gosec
	fakeSigningSecret = "fake-signing-secret"
	fakeAppToken      = "xapp-fake-app-token" //nolint:gosec
	fakeBotID         = "B12345"
	fakeBotUserID     = "U12345"
)

// CreateTestSlackEventMiddlewareArgs creates a SlackEventMiddlewareArgs for testing
func CreateTestSlackEventMiddlewareArgs(baseArgs types.AllMiddlewareArgs, eventData, bodyData map[string]interface{}) types.SlackEventMiddlewareArgs {
	// Parse into strongly typed structures
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)
	parsedEnvelope, _ := helpers.ParseEventEnvelope(bodyData)

	return types.SlackEventMiddlewareArgs{
		AllMiddlewareArgs: baseArgs,
		Event:             parsedEvent,
		Body:              parsedEnvelope,
	}
}

// CreateTestSlackEventFromMap creates a SlackEvent from raw map data for testing
func CreateTestSlackEventFromMap(eventData map[string]interface{}) types.SlackEvent {
	parsedEvent, _ := helpers.ParseSlackEvent(eventData)
	return parsedEvent
}

// CreateTestEventEnvelopeFromMap creates an EventEnvelope from raw map data for testing
func CreateTestEventEnvelopeFromMap(bodyData map[string]interface{}) types.EventEnvelope {
	parsedEnvelope, _ := helpers.ParseEventEnvelope(bodyData)
	return parsedEnvelope
}

// ExtractRawEventData extracts raw map data from a strongly typed SlackEvent for testing
func ExtractRawEventData(event types.SlackEvent) (map[string]interface{}, bool) {
	if genericEvent, ok := event.(*helpers.GenericSlackEvent); ok {
		return genericEvent.RawData, true
	}
	// Fallback: marshal/unmarshal
	rawData, err := helpers.ExtractRawDataFromSlackEvent(event)
	if err != nil {
		return nil, false
	}
	return rawData, true
}

// ExtractRawActionData extracts raw map data from a strongly typed SlackAction for testing
func ExtractRawActionData(action types.SlackAction) (map[string]interface{}, bool) {
	rawData, err := helpers.ExtractRawDataFromSlackAction(action)
	if err != nil {
		return nil, false
	}
	return rawData, true
}

// ExtractRawShortcutData extracts raw map data from a strongly typed SlackShortcut for testing
func ExtractRawShortcutData(shortcut types.SlackShortcut) (map[string]interface{}, bool) {
	rawData, err := helpers.ExtractRawDataFromSlackShortcut(shortcut)
	if err != nil {
		return nil, false
	}
	return rawData, true
}

// ExtractRawViewData extracts raw map data from a strongly typed SlackView for testing
func ExtractRawViewData(view types.SlackView) (map[string]interface{}, bool) {
	rawData, err := helpers.ExtractRawDataFromSlackView(view)
	if err != nil {
		return nil, false
	}
	return rawData, true
}

// CreateTestEventData creates a basic event data map
func CreateTestEventData(eventType string, additionalFields map[string]interface{}) map[string]interface{} {
	eventData := map[string]interface{}{
		"type": eventType,
	}

	// Add additional fields
	for k, v := range additionalFields {
		eventData[k] = v
	}

	return eventData
}

// CreateTestBodyData creates a basic body data map for events
func CreateTestBodyData(eventData map[string]interface{}, additionalFields map[string]interface{}) map[string]interface{} {
	bodyData := map[string]interface{}{
		"event":      eventData,
		"team_id":    "T123",
		"api_app_id": "A123",
		"type":       "event_callback",
	}

	// Add additional fields
	for k, v := range additionalFields {
		bodyData[k] = v
	}

	return bodyData
}
