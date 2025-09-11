# Socket Mode with OAuth Example

This example demonstrates how to use Socket Mode combined with OAuth in Bolt for Go. This setup allows for easy local development while still supporting OAuth installation flow.

## Features Demonstrated

- Socket Mode connection for real-time events
- OAuth installation flow support
- Multiple OAuth scopes configuration
- Minimal setup for development and testing

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_APP_TOKEN=xapp-your-app-token
   export SLACK_SIGNING_SECRET=your-signing-secret
   export SLACK_CLIENT_ID=your-client-id
   export SLACK_CLIENT_SECRET=your-client-secret
   export SLACK_STATE_SECRET=your-state-secret
   export PORT=3000  # optional, for OAuth endpoints
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the app:
   ```bash
   go run main.go
   ```

## OAuth Configuration

### Slack App Settings

1. **Socket Mode**: Enable Socket Mode in your app settings
2. **App Token**: Generate an app-level token with `connections:write` scope
3. **OAuth Scopes**: Configure the following bot scopes:
   - `channels:history` - to read channel history
   - `chat:write` - to send messages
   - `commands` - to handle slash commands

### OAuth & Permissions

1. Set Redirect URLs to include:
   ```
   http://localhost:3000/slack/oauth_redirect
   ```

2. The app will automatically handle:
   - Installation at `/slack/install`
   - OAuth callback at `/slack/oauth_redirect`

## How It Works

1. **Socket Mode**: Provides real-time bidirectional communication with Slack
2. **OAuth Flow**: Handles app installation and token management
3. **Hybrid Approach**: Socket Mode for events, HTTP for OAuth endpoints

## Benefits of This Setup

- **Development Friendly**: No need to expose localhost to the internet
- **OAuth Support**: Still supports proper app distribution
- **Real-time Events**: Instant event delivery via WebSocket
- **Secure**: No webhook URLs to secure

## Installation Flow

1. Navigate to `http://localhost:3000/slack/install`
2. Complete OAuth flow in Slack
3. App receives installation and starts listening for events via Socket Mode

## Production Considerations

- Socket Mode is great for development but consider HTTP receivers for production
- Implement proper installation store (database instead of memory)
- Use secure secrets management
- Consider scaling implications of WebSocket connections
- Monitor connection health and implement reconnection logic

## Scopes Explanation

- **channels:history**: Read messages and content from public channels
- **chat:write**: Send messages as the bot user
- **commands**: Receive and respond to slash commands

## Troubleshooting

- Ensure Socket Mode is enabled in your Slack app
- Verify app token has `connections:write` scope
- Check that all environment variables are set correctly
- Monitor logs for connection status and errors
