# Event Types Demo

This example demonstrates how to use the new typed event constants in Bolt Go, providing better type safety and preventing common typos when registering event listeners.

## Features Demonstrated

1. **Typed Event Constants** - Using `types.EventTypeAppMention` instead of raw strings
2. **Custom Event Types** - Creating custom event types with `types.SlackEventType("custom")`
3. **Multiple Event Handlers** - Registering handlers for multiple event types efficiently
4. **Event Type Validation** - Checking if event types are valid
5. **Available Event Types** - Listing all supported Slack event types

## Benefits of Typed Event Constants

- **Type Safety**: Compile-time checking prevents typos in event type names
- **IDE Support**: Better autocomplete and IntelliSense
- **Flexibility**: Easy to create custom event types with `types.SlackEventType("custom")`
- **Documentation**: Self-documenting code with clear event type names
- **Validation**: Built-in validation to check if event types are supported
- **Clean API**: Single type parameter eliminates confusion

## Usage

```go
// ✅ Use predefined typed constants
boltApp.Event(types.EventTypeAppMention, handler)

// ✅ Create custom event types when needed
boltApp.Event(types.SlackEventType("my_custom_event"), handler)

// ❌ Raw strings no longer accepted (compile error)
// boltApp.Event("app_mention", handler) // This won't compile
```

## Available Event Types

The following event type constants are available:

- `types.EventTypeMessage` - Message events
- `types.EventTypeAppMention` - App mention events  
- `types.EventTypeReactionAdded` - Reaction added events
- `types.EventTypeChannelCreated` - Channel creation events
- `types.EventTypeFunctionExecuted` - Function execution events
- And many more... (see `pkg/types/event_types.go` for the complete list)

## Running the Example

1. Set your environment variables:
   ```bash
   export SLACK_BOT_TOKEN=xoxb-your-bot-token
   export SLACK_APP_TOKEN=xapp-your-app-token
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

3. Try mentioning your bot or sending it a direct message to see the different event handling approaches in action.

## Event Type Validation

You can validate event types programmatically:

```go
// Check if an event type is valid
isValid := types.EventTypeMessage.IsValid() // true

// Get all available event types
allTypes := types.AllEventTypes()
fmt.Printf("Total event types: %d\n", len(allTypes))
```
