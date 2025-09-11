# Socket Mode Example

This example demonstrates how to use Bolt for Go with Socket Mode, which allows your app to connect to Slack via WebSockets instead of HTTP.

## Features Demonstrated

- Socket Mode connection setup
- App Home publishing
- Message and Global shortcuts
- App mention handling
- Message listeners
- Button interactions
- Slash command handling
- Modal views

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

3. Run the app:
   ```bash
   go run main.go
   ```

## Required Scopes

Make sure your Slack app has these scopes:
- `app_mentions:read` - to receive app mentions
- `chat:write` - to send messages
- `channels:read` - to read channel messages
- `commands` - for slash commands

## Usage

- Mention your bot in a channel to see the app mention handler
- Send "hello" in a channel to trigger the message listener
- Use the `/socketslash` command to test slash command handling
- Set up shortcuts in your app config to test shortcut handlers
- Visit your app's Home tab to see the App Home

## Key Benefits of Socket Mode

- No need to expose your app to the internet during development
- Real-time bidirectional communication
- Simpler local development setup
- No webhook URL configuration required
