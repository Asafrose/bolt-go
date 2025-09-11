# OAuth Example

This example demonstrates how to set up OAuth flow with Bolt for Go, allowing users to install your app to their Slack workspaces.

## Features Demonstrated

- OAuth installation flow setup
- Custom installation store implementation
- In-memory database simulation for installations
- Enterprise and single-team installation support
- Installation data persistence and retrieval

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_CLIENT_ID=your-client-id
   export SLACK_CLIENT_SECRET=your-client-secret
   export SLACK_SIGNING_SECRET=your-signing-secret
   export SLACK_STATE_SECRET=your-state-secret
   export PORT=3000  # optional, defaults to 3000
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

1. Navigate to `http://localhost:3000/slack/install` to start the installation process
2. You'll be redirected to Slack's authorization page
3. After authorization, you'll be redirected back to your app
4. The installation data will be stored in the in-memory database

## Production Considerations

- Replace the in-memory database with a persistent storage solution (PostgreSQL, MongoDB, etc.)
- Implement proper error handling and logging
- Use secure secrets management
- Set up proper HTTPS endpoints for production
- Consider implementing installation validation and cleanup

## Required App Configuration

In your Slack app configuration:
- Set OAuth & Permissions > Redirect URLs to include `http://localhost:3000/slack/oauth_redirect`
- Add the required scopes under OAuth & Permissions > Scopes
- Enable public distribution if you want others to install your app

## Key Components

- **Installation Store**: Handles storing, retrieving, and deleting installation data
- **OAuth Flow**: Manages the authorization process between Slack and your app
- **State Secret**: Used to prevent CSRF attacks during OAuth flow
