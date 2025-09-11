# OAuth HTTP Receiver Example

This example demonstrates how to use OAuth with an HTTP receiver in Bolt for Go. This is equivalent to the Express receiver in the JavaScript version, allowing you to handle both Slack events and custom HTTP routes.

## Features Demonstrated

- HTTP receiver with OAuth support
- File-based installation store
- Custom HTTP routes alongside Slack event handling
- OAuth installation flow
- Event listening with client access

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_CLIENT_ID=your-client-id
   export SLACK_CLIENT_SECRET=your-client-secret
   export SLACK_SIGNING_SECRET=your-signing-secret
   export SLACK_STATE_SECRET=your-state-secret
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the app:
   ```bash
   go run main.go
   ```

## OAuth Flow

1. Navigate to `http://localhost:3000/slack/install` to start installation
2. Complete the OAuth flow in Slack
3. Installation data will be saved to `installations.json`

## Custom Routes

The example includes a custom route at `/secret-page` that demonstrates how to add your own HTTP handlers alongside Slack event handling.

Visit `http://localhost:3000/secret-page` to see the custom route in action.

## File Installation Store

This example uses a file-based installation store that saves installation data to a JSON file. In production, you should use a proper database.

The installation data includes:
- Team information
- Bot token
- User tokens (if user scopes are requested)
- Installation metadata

## Key Benefits

- **Unified Server**: Handle both Slack events and web requests in one app
- **OAuth Support**: Built-in OAuth flow handling
- **Custom Routes**: Add your own HTTP endpoints
- **Persistent Storage**: File-based installation store for development

## Production Considerations

- Replace file-based storage with a database
- Use HTTPS in production
- Implement proper error handling
- Add authentication for custom routes
- Use secure secrets management
- Consider rate limiting and request validation

## Slack App Configuration

1. Set OAuth & Permissions > Redirect URLs to:
   ```
   http://localhost:3000/slack/oauth_redirect
   ```

2. Add required scopes under OAuth & Permissions > Scopes

3. Subscribe to events you want to listen for

4. Set Event Subscriptions > Request URL to:
   ```
   http://localhost:3000/slack/events
   ```
