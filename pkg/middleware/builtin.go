package middleware

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/types"
)

// OnlyActions filters to only process action events
func OnlyActions(args types.AllMiddlewareArgs) error {
	// Check if this is an action event by looking at the context
	if args.Context != nil && args.Context.Custom != nil {
		if eventType, exists := args.Context.Custom["eventType"]; exists {
			if eventTypeVal, ok := eventType.(helpers.IncomingEventType); ok {
				if eventTypeVal == helpers.IncomingEventTypeAction {
					return args.Next()
				}
			}
		}
	}
	return nil // Skip if not an action
}

// OnlyShortcuts filters to only process shortcut events
func OnlyShortcuts(args types.AllMiddlewareArgs) error {
	typeAndConv := helpers.GetTypeAndConversation([]byte{}) // We'll need to pass body somehow
	if typeAndConv.Type != nil && *typeAndConv.Type == helpers.IncomingEventTypeShortcut {
		return args.Next()
	}
	return nil
}

// OnlyCommands filters to only process command events
func OnlyCommands(args types.AllMiddlewareArgs) error {
	// Check if this is a command event by looking at the context
	if args.Context != nil && args.Context.Custom != nil {
		if eventType, exists := args.Context.Custom["eventType"]; exists {
			if eventTypeVal, ok := eventType.(helpers.IncomingEventType); ok {
				if eventTypeVal == helpers.IncomingEventTypeCommand {
					return args.Next()
				}
			}
		}
	}
	return nil // Skip if not a command
}

// OnlyEvents filters to only process events
func OnlyEvents(args types.AllMiddlewareArgs) error {
	// Check if this is an event by looking at the context
	if args.Context != nil && args.Context.Custom != nil {
		if eventType, exists := args.Context.Custom["eventType"]; exists {
			if eventTypeVal, ok := eventType.(helpers.IncomingEventType); ok {
				if eventTypeVal == helpers.IncomingEventTypeEvent {
					return args.Next()
				}
			}
		}
	}
	return nil // Skip if not an event
}

// OnlyOptions filters to only process options requests
func OnlyOptions(args types.AllMiddlewareArgs) error {
	typeAndConv := helpers.GetTypeAndConversation([]byte{}) // We'll need to pass body somehow
	if typeAndConv.Type != nil && *typeAndConv.Type == helpers.IncomingEventTypeOptions {
		return args.Next()
	}
	return nil
}

// OnlyViewActions filters to only process view actions
func OnlyViewActions(args types.AllMiddlewareArgs) error {
	typeAndConv := helpers.GetTypeAndConversation([]byte{}) // We'll need to pass body somehow
	if typeAndConv.Type != nil && *typeAndConv.Type == helpers.IncomingEventTypeViewAction {
		return args.Next()
	}
	return nil
}

// MatchEventType creates middleware that matches specific event types (string or RegExp)
func MatchEventType(pattern interface{}) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Only process event middleware args
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				// Extract event type from event data
				var eventMap map[string]interface{}
				if genericEvent, ok := eventArgs.Event.(*helpers.GenericSlackEvent); ok {
					eventMap = genericEvent.RawData
				} else {
					// Fallback: try to marshal/unmarshal to get raw data
					if eventBytes, err := json.Marshal(eventArgs.Event); err == nil {
						json.Unmarshal(eventBytes, &eventMap)
					}
				}

				if eventMap != nil {
					if actualEventType, exists := eventMap["type"]; exists {
						if typeStr, ok := actualEventType.(string); ok {
							// Match using pattern (string or RegExp)
							if helpers.MatchesPattern(typeStr, pattern) {
								// For RegExp patterns, store matches in context
								if regexPattern, ok := pattern.(*regexp.Regexp); ok {
									if matches := regexPattern.FindStringSubmatch(typeStr); matches != nil {
										if args.Context.Custom == nil {
											args.Context.Custom = make(map[string]interface{})
										}
										args.Context.Custom["matches"] = matches
									}
								} else if regexPattern, ok := pattern.(regexp.Regexp); ok {
									if matches := regexPattern.FindStringSubmatch(typeStr); matches != nil {
										if args.Context.Custom == nil {
											args.Context.Custom = make(map[string]interface{})
										}
										args.Context.Custom["matches"] = matches
									}
								}
								return args.Next()
							}
						}
					}
				}
			}
		}

		// Event type doesn't match, skip processing
		return nil
	}
}

// MatchCommandName creates middleware that matches command names
func MatchCommandName(pattern interface{}) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Only process command middleware args
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if commandArgs, ok := middlewareArgs.(types.SlackCommandMiddlewareArgs); ok {
				commandName := commandArgs.Command.Command

				// Match using pattern (string or RegExp)
				if helpers.MatchesPattern(commandName, pattern) {
					return args.Next()
				}
			}
		}

		// Command doesn't match, skip processing
		return nil
	}
}

// MatchConstraints creates middleware that matches action constraints
func MatchConstraints(constraints types.ActionConstraints) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// This would need access to the actual action payload
		// For now, always proceed
		return args.Next()
	}
}

// MatchMessage creates middleware that matches message patterns
func MatchMessage(pattern interface{}) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Only process message events
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				if eventArgs.Message != nil && eventArgs.Message.Text != "" {
					text := eventArgs.Message.Text

					// Match using pattern (string or RegExp)
					if helpers.MatchesPattern(text, pattern) {
						// For RegExp patterns, store matches in context
						if regexPattern, ok := pattern.(*regexp.Regexp); ok {
							if matches := regexPattern.FindStringSubmatch(text); matches != nil {
								if args.Context.Custom == nil {
									args.Context.Custom = make(map[string]interface{})
								}
								args.Context.Custom["matches"] = matches
							}
						} else if regexPattern, ok := pattern.(regexp.Regexp); ok {
							if matches := regexPattern.FindStringSubmatch(text); matches != nil {
								if args.Context.Custom == nil {
									args.Context.Custom = make(map[string]interface{})
								}
								args.Context.Custom["matches"] = matches
							}
						}

						return args.Next()
					}
				}
			}
		}

		// Message doesn't match, skip processing
		return nil
	}
}

// IgnoreSelf creates middleware that ignores events from the bot itself
func IgnoreSelf() types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		botID := args.Context.BotID
		botUserID := args.Context.BotUserID

		// Try to extract middleware args stored in context
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				// Check for bot messages first
				if eventArgs.Message != nil {
					// Look for an event that is identified as a bot message from the same bot ID as this app
					if eventArgs.Message.SubType == "bot_message" && botID != nil {
						// Check both the embedded BotID (string) and the additional BotID (*string) field
						messageBotID := ""
						if eventArgs.Message.MessageEvent.BotID != "" {
							messageBotID = eventArgs.Message.MessageEvent.BotID
						} else if eventArgs.Message.BotID != nil {
							messageBotID = *eventArgs.Message.BotID
						}

						if messageBotID != "" && messageBotID == *botID {
							return nil // Skip processing
						}
					}
				}

				// Check for regular events with user ID matching bot user ID
				// However, some events still must be fired, because they can make sense
				eventsWhichShouldBeKept := []string{"member_joined_channel", "member_left_channel"}

				if botUserID != nil {
					var eventMap map[string]interface{}
					if genericEvent, ok := eventArgs.Event.(*helpers.GenericSlackEvent); ok {
						eventMap = genericEvent.RawData
					} else {
						// Fallback: try to marshal/unmarshal to get raw data
						if eventBytes, err := json.Marshal(eventArgs.Event); err == nil {
							json.Unmarshal(eventBytes, &eventMap)
						}
					}

					if eventMap != nil {
						if eventUserID := ExtractUserFromEvent(eventMap); eventUserID != nil && *eventUserID == *botUserID {
							// Check if this event type should be kept
							if eventType, exists := eventMap["type"]; exists {
								if eventTypeStr, ok := eventType.(string); ok {
									for _, keepEventType := range eventsWhichShouldBeKept {
										if eventTypeStr == keepEventType {
											// This event should be kept, continue processing
											return args.Next()
										}
									}
								}
							}
							// Event user matches bot user and it's not an exception, skip processing
							return nil
						}
					}
				}
			}
		}

		// If all the previous checks didn't skip this message, then it's okay to resume to next
		return args.Next()
	}
}

// DirectMention creates middleware that filters messages that don't start with @mention of the bot
func DirectMention() types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Get bot user ID from context
		if args.Context == nil || args.Context.BotUserID == nil {
			// Cannot perform direct mention matching without bot user ID
			return errors.NewContextMissingPropertyError("botUserId", "Cannot match direct mentions of the app without a bot user ID. Ensure authorize callback returns a botUserId.")
		}

		// Only process message events
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				if eventArgs.Message != nil && eventArgs.Message.Text != "" {
					text := strings.TrimSpace(eventArgs.Message.Text)

					// Check if message starts with @mention of the bot
					mentionPattern := regexp.MustCompile(`^<@([^>|]+)(?:\|([^>]+))?>`)
					matches := mentionPattern.FindStringSubmatch(text)

					if len(matches) >= 2 && matches[1] == *args.Context.BotUserID {
						// Message starts with bot mention, continue processing
						return args.Next()
					}
				}
			}
		}

		// Not a direct mention, skip processing
		return nil
	}
}

// Helper functions for pattern matching

// IsBlockPayload checks if payload is a block action or suggestion
func IsBlockPayload(payload interface{}) bool {
	if payloadMap, ok := payload.(map[string]interface{}); ok {
		if actionID, exists := payloadMap["action_id"]; exists {
			return actionID != nil
		}
	}
	return false
}

// IsCallbackIdentifiedBody checks if body has callback_id
func IsCallbackIdentifiedBody(body interface{}) bool {
	if bodyMap, ok := body.(map[string]interface{}); ok {
		if callbackID, exists := bodyMap["callback_id"]; exists {
			return callbackID != nil
		}
	}
	return false
}

// IsViewBody checks if body contains a view
func IsViewBody(body interface{}) bool {
	if bodyMap, ok := body.(map[string]interface{}); ok {
		if view, exists := bodyMap["view"]; exists {
			return view != nil
		}
	}
	return false
}

// MatchesRegexPattern checks if text matches a regex pattern
func MatchesRegexPattern(text string, pattern *regexp.Regexp) bool {
	if pattern == nil {
		return true
	}
	return pattern.MatchString(text)
}

// MatchesStringPattern checks if text matches a string pattern
func MatchesStringPattern(text, pattern string) bool {
	if pattern == "" {
		return true
	}
	return strings.Contains(text, pattern)
}

// ExtractChannelFromEvent extracts channel ID from event data
func ExtractChannelFromEvent(eventData interface{}) *string {
	if eventMap, ok := eventData.(map[string]interface{}); ok {
		// Try different channel field variations
		if channel, exists := eventMap["channel"]; exists {
			if channelStr, ok := channel.(string); ok {
				return &channelStr
			} else if channelMap, ok := channel.(map[string]interface{}); ok {
				if id, exists := channelMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						return &idStr
					}
				}
			}
		}

		if channelID, exists := eventMap["channel_id"]; exists {
			if channelIDStr, ok := channelID.(string); ok {
				return &channelIDStr
			}
		}
	}
	return nil
}

// ExtractUserFromEvent extracts user ID from event data
func ExtractUserFromEvent(eventData interface{}) *string {
	if eventMap, ok := eventData.(map[string]interface{}); ok {
		if user, exists := eventMap["user"]; exists {
			if userStr, ok := user.(string); ok {
				return &userStr
			} else if userMap, ok := user.(map[string]interface{}); ok {
				if id, exists := userMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						return &idStr
					}
				}
			}
		}

		if userID, exists := eventMap["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				return &userIDStr
			}
		}
	}
	return nil
}

// CreateMiddlewareChain creates a chain of middleware functions
func CreateMiddlewareChain(middlewares ...types.Middleware[types.AllMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		index := 0

		var next types.NextFn
		next = func() error {
			if index >= len(middlewares) {
				return args.Next()
			}

			currentMiddleware := middlewares[index]
			index++

			// Create new args with updated Next function
			newArgs := types.AllMiddlewareArgs{
				Context: args.Context,
				Logger:  args.Logger,
				Client:  args.Client,
				Next:    next,
			}

			return currentMiddleware(newArgs)
		}

		return next()
	}
}

// ProcessMessageEvent processes message events for pattern matching
func ProcessMessageEvent(body []byte, pattern interface{}) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}

	if event, exists := parsed["event"]; exists {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if text, exists := eventMap["text"]; exists {
				if textStr, ok := text.(string); ok {
					return helpers.MatchesPattern(textStr, pattern)
				}
			}
		}
	}

	return false
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

// ExtractEnterpriseID extracts enterprise ID from the body
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

// Subtype creates middleware that filters message events by subtype
func Subtype(subtype string) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Only process message events
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			if eventArgs, ok := middlewareArgs.(types.SlackEventMiddlewareArgs); ok {
				if eventArgs.Message != nil {
					// Check if message has the matching subtype
					if eventArgs.Message.SubType == subtype {
						return args.Next()
					}
				}
			}
		}

		// Subtype doesn't match, skip processing
		return nil
	}
}

// SlackEventMiddlewareArgsOptions represents options for event middleware
type SlackEventMiddlewareArgsOptions struct {
	AutoAcknowledge bool `json:"auto_acknowledge"`
}

// AutoAcknowledge is middleware that auto acknowledges the request received
func AutoAcknowledge(args types.AllMiddlewareArgs) error {
	// Try to extract middleware args stored in context to find ack function
	if args.Context != nil && args.Context.Custom != nil {
		if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
			// Check for different types of middleware args that have ack functions
			switch typedArgs := middlewareArgs.(type) {
			case types.SlackActionMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackCommandMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackEventMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackShortcutMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackOptionsMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackViewMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			case types.SlackCustomFunctionMiddlewareArgs:
				if typedArgs.Ack != nil {
					if err := typedArgs.Ack(nil); err != nil {
						return err
					}
				}
			}
		}
	}

	return args.Next()
}

// MatchCallbackId creates middleware that matches function callback IDs
func MatchCallbackId(callbackId string) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Only process custom function middleware args
		if args.Context != nil && args.Context.Custom != nil {
			if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
				if customFunctionArgs, ok := middlewareArgs.(types.SlackCustomFunctionMiddlewareArgs); ok {
					// Extract callback_id from payload
					if payloadMap, ok := customFunctionArgs.Payload.(map[string]interface{}); ok {
						if functionMap, exists := payloadMap["function"]; exists {
							if functionData, ok := functionMap.(map[string]interface{}); ok {
								if actualCallbackId, exists := functionData["callback_id"]; exists {
									if callbackIdStr, ok := actualCallbackId.(string); ok {
										if callbackIdStr == callbackId {
											return args.Next()
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Callback ID doesn't match, skip processing
		return nil
	}
}

// IsSlackEventMiddlewareArgsOptions checks if the given interface is SlackEventMiddlewareArgsOptions
func IsSlackEventMiddlewareArgsOptions(optionOrListener interface{}) bool {
	if optionOrListener == nil {
		return false
	}

	// Check if it's a function (middleware)
	if _, isFunc := optionOrListener.(func(types.AllMiddlewareArgs) error); isFunc {
		return false
	}

	// Check if it's a struct/map with autoAcknowledge field
	switch v := optionOrListener.(type) {
	case SlackEventMiddlewareArgsOptions:
		return true
	case map[string]interface{}:
		_, hasAutoAck := v["autoAcknowledge"]
		return hasAutoAck
	case map[string]bool:
		_, hasAutoAck := v["autoAcknowledge"]
		return hasAutoAck
	default:
		return false
	}
}
