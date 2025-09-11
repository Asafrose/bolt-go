# Message Metadata Example

This example demonstrates how to use Message Metadata with Bolt for Go. Message Metadata allows you to attach structured data to messages and listen for events when metadata is posted.

## Features Demonstrated

- Posting messages with metadata using slash commands
- Listening for `message_metadata_posted` events
- Responding in threads when metadata events occur
- JSON marshaling for metadata display

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_BOT_TOKEN=xoxb-your-bot-token
   export SLACK_APP_TOKEN=xapp-your-app-token
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Configure your Slack app:
   - Enable Socket Mode
   - Subscribe to the `message_metadata_posted` event
   - Create a slash command `/post`
   - Add required scopes: `chat:write`, `commands`

4. Run the app:
   ```bash
   go run main.go
   ```

## Usage

1. Use the `/post` slash command in any channel
2. The app will post a message with metadata attached
3. Slack will fire a `message_metadata_posted` event
4. The app will respond in a thread showing the metadata that was posted

## Message Metadata

Message Metadata is a feature that allows you to:
- Attach structured data to messages
- Listen for events when messages with metadata are posted
- Build workflows that respond to metadata events
- Store additional context with messages

## Event Flow

1. User runs `/post` command
2. App acknowledges and posts message with metadata
3. Slack fires `message_metadata_posted` event
4. App receives event and responds in thread with metadata details

## Use Cases

- Workflow triggers based on message content
- Tracking message context and state
- Building approval workflows
- Connecting messages to external systems
- Audit trails and message tracking
