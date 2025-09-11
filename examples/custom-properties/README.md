# Custom Properties Example

This example demonstrates how to use custom properties extractors with different receivers in Bolt for Go. Custom properties allow you to add additional context to incoming requests.

## Features Demonstrated

- HTTP receiver with custom properties extractor
- Socket Mode receiver with custom properties extractor  
- Custom error handlers for HTTP receiver
- Middleware that accesses custom properties
- Deferred initialization pattern

## Examples Included

### HTTP Receiver (`http/main.go`)
- Extracts HTTP headers and custom data
- Custom dispatch error handler
- Custom process event error handler
- Custom unhandled request handler
- Custom timeout configuration

### Socket Mode Receiver (`socket-mode/main.go`)
- Extracts Socket Mode payload information
- Adds custom properties to Socket Mode events

## Setup

1. Set your environment variables:
   ```bash
   # For HTTP receiver
   export SLACK_BOT_TOKEN=xoxb-your-bot-token
   export SLACK_SIGNING_SECRET=your-signing-secret
   export PORT=3000  # optional

   # For Socket Mode receiver (additional)
   export SLACK_APP_TOKEN=xapp-your-app-token
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### Run HTTP Receiver Example:
```bash
cd http
go run main.go
```

### Run Socket Mode Receiver Example:
```bash
cd socket-mode
go run main.go
```

## Custom Properties Use Cases

- **Request Tracking**: Add request IDs or correlation IDs
- **Authentication Context**: Include user authentication details
- **Rate Limiting**: Add rate limiting information
- **A/B Testing**: Include experiment flags
- **Debugging**: Add debug information for troubleshooting
- **Analytics**: Include tracking data for metrics

## Error Handling

The HTTP receiver example demonstrates custom error handlers:

- **DispatchErrorHandler**: Handles errors during request dispatch
- **ProcessEventErrorHandler**: Handles errors during event processing
- **UnhandledRequestHandler**: Handles requests that timeout or aren't processed

## Key Benefits

- **Enhanced Context**: Access to additional request information in handlers
- **Flexible Integration**: Easy integration with existing middleware
- **Custom Error Handling**: Fine-grained control over error responses
- **Debugging Support**: Better visibility into request processing
