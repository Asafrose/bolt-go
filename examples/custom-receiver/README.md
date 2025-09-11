# Custom Receiver Examples

This example demonstrates how to create custom receivers for Bolt for Go using popular Go web frameworks. Custom receivers allow you to integrate Bolt with your preferred web framework while maintaining full control over routing and middleware.

## Examples Included

### Gin Receiver (`gin-receiver/main.go`)
Demonstrates creating a custom receiver using the [Gin Web Framework](https://github.com/gin-gonic/gin):
- Fast HTTP router with middleware support
- JSON binding and validation
- Template rendering support
- Comprehensive error handling

### Echo Receiver (`echo-receiver/main.go`)
Demonstrates creating a custom receiver using the [Echo Web Framework](https://github.com/labstack/echo):
- High-performance HTTP router
- Built-in middleware for logging, recovery, etc.
- WebSocket support
- HTTP/2 support

## Features Demonstrated

- Custom receiver implementation patterns
- OAuth flow integration with web frameworks
- Custom routing alongside Slack event handling
- Framework-specific middleware usage
- Template rendering for OAuth pages
- Health check endpoints

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_SIGNING_SECRET=your-signing-secret
   export SLACK_CLIENT_ID=your-client-id
   export SLACK_CLIENT_SECRET=your-client-secret
   export PORT=3000  # optional
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### Run Gin Receiver Example:
```bash
cd gin-receiver
go run main.go
```

### Run Echo Receiver Example:
```bash
cd echo-receiver
go run main.go
```

## Available Endpoints

Both examples provide the following endpoints:

- `GET /` - Redirects to `/slack/install`
- `GET /slack/install` - OAuth installation page
- `GET /slack/oauth_redirect` - OAuth callback handler
- `POST /slack/events` - Slack events webhook
- `GET /health` - Health check endpoint (Echo example)

## Custom Receiver Interface

To create a custom receiver, implement the following interface:

```go
type Receiver interface {
    Init(app *app.App) error
    Start(port int) error
    ProcessEvent(ctx context.Context, event *types.ReceiverEvent) error
}
```

## Key Benefits

- **Framework Freedom**: Use your preferred Go web framework
- **Middleware Control**: Full control over HTTP middleware stack
- **Custom Routing**: Add your own API endpoints alongside Slack handlers
- **Performance Optimization**: Framework-specific optimizations
- **Integration**: Easy integration with existing Go web applications

## Production Considerations

- Implement proper request signature verification
- Add comprehensive error handling and logging
- Use production-ready OAuth flow implementation
- Implement proper installation store (database)
- Add rate limiting and request validation
- Use HTTPS in production
- Implement health checks and monitoring

## Framework Comparison

| Feature | Gin | Echo |
|---------|-----|------|
| Performance | Very Fast | Very Fast |
| Middleware | Rich ecosystem | Built-in + extensible |
| Learning Curve | Easy | Easy |
| WebSocket | Via gorilla/websocket | Built-in |
| Template Engine | Built-in | External |
| JSON Handling | Excellent | Excellent |

## Extending the Examples

You can extend these examples by:

- Adding authentication middleware
- Implementing database integration
- Adding API versioning
- Implementing rate limiting
- Adding metrics and monitoring
- Creating custom middleware for Slack-specific features
