# Bolt for Go Examples

This directory contains comprehensive examples demonstrating various features and deployment patterns of the Bolt framework for Go. Each example is a complete, runnable application that showcases specific functionality.

## 🚀 Getting Started

### [Getting Started](./getting-started/)
Basic Slack app setup with message listeners, button actions, and middleware. Perfect for beginners.

**Features:** Basic app setup, message patterns, block kit, button interactions

## 🔌 Connection Types

### [Socket Mode](./socket-mode/)
Real-time WebSocket connection to Slack with comprehensive event handling.

**Features:** Socket Mode, app mentions, shortcuts, modals, slash commands

### [HTTP Receiver](./oauth-http-receiver/)
HTTP-based receiver with OAuth support for production deployments.

**Features:** HTTP receiver, OAuth flow, custom routes, file installation store

## 🔐 Authentication & OAuth

### [OAuth](./oauth/)
Complete OAuth implementation with custom installation store.

**Features:** OAuth flow, installation management, enterprise support

### [Socket Mode + OAuth](./socket-mode-oauth/)
Hybrid approach combining Socket Mode with OAuth installation flow.

**Features:** Socket Mode, OAuth installation, development-friendly setup

## ☁️ Cloud Deployments

### [AWS Lambda](./aws-lambda/)
Serverless deployment using AWS Lambda and API Gateway.

**Features:** Lambda receiver, serverless architecture, cross-compilation

### [Heroku](./heroku/)
Platform-as-a-Service deployment with Heroku.

**Features:** Heroku deployment, Procfile, environment configuration

## 🔧 Advanced Features

### [Message Metadata](./message-metadata/)
Working with Slack's Message Metadata feature for workflow automation.

**Features:** Message metadata, event listening, threaded responses

### [Custom Properties](./custom-properties/)
Adding custom context to requests with different receiver types.

**Features:** Custom properties extraction, HTTP/Socket Mode, error handling

### [Custom Receivers](./custom-receiver/)
Building custom receivers with popular Go web frameworks.

**Features:** Gin receiver, Echo receiver, custom routing, middleware integration

## 📋 Quick Reference

| Example | Connection | OAuth | Deployment | Complexity |
|---------|------------|-------|------------|------------|
| Getting Started | HTTP | ❌ | Local | Beginner |
| Socket Mode | WebSocket | ❌ | Local | Beginner |
| OAuth | HTTP | ✅ | Local/Cloud | Intermediate |
| Socket Mode + OAuth | WebSocket | ✅ | Local | Intermediate |
| AWS Lambda | HTTP | ❌ | Serverless | Intermediate |
| Heroku | HTTP | ❌ | PaaS | Beginner |
| Message Metadata | WebSocket | ❌ | Local | Intermediate |
| Custom Properties | Both | ❌ | Local | Advanced |
| Custom Receivers | HTTP | ✅ | Local/Cloud | Advanced |

## 🛠️ Setup Requirements

### Environment Variables

Most examples require these environment variables:

```bash
# Required for all examples
export SLACK_BOT_TOKEN=xoxb-your-bot-token
export SLACK_SIGNING_SECRET=your-signing-secret

# Required for Socket Mode examples
export SLACK_APP_TOKEN=xapp-your-app-token

# Required for OAuth examples
export SLACK_CLIENT_ID=your-client-id
export SLACK_CLIENT_SECRET=your-client-secret
export SLACK_STATE_SECRET=your-state-secret

# Optional
export PORT=3000
```

### Slack App Configuration

1. **Create a Slack App** at [api.slack.com/apps](https://api.slack.com/apps)
2. **Configure OAuth & Permissions**:
   - Add bot scopes: `chat:write`, `app_mentions:read`, etc.
   - Set redirect URLs for OAuth examples
3. **Configure Event Subscriptions**:
   - Set request URL for HTTP examples
   - Subscribe to bot events
4. **Enable Socket Mode** (for Socket Mode examples):
   - Generate app-level token with `connections:write` scope

## 🏃‍♂️ Running Examples

Each example is self-contained with its own `go.mod` file:

```bash
# Navigate to any example directory
cd getting-started

# Install dependencies
go mod tidy

# Run the example
go run main.go
```

## 📚 Learning Path

### Beginner
1. Start with [Getting Started](./getting-started/) for basic concepts
2. Try [Socket Mode](./socket-mode/) for local development
3. Explore [Heroku](./heroku/) for simple deployment

### Intermediate
4. Learn [OAuth](./oauth/) for app distribution
5. Try [AWS Lambda](./aws-lambda/) for serverless
6. Experiment with [Message Metadata](./message-metadata/)

### Advanced
7. Explore [Custom Properties](./custom-properties/) for advanced context
8. Build [Custom Receivers](./custom-receiver/) for framework integration
9. Combine patterns for complex applications

## 🤝 Contributing

Found an issue or want to add an example? Contributions are welcome!

1. Ensure your example follows the established patterns
2. Include comprehensive README documentation
3. Add proper error handling and logging
4. Test with actual Slack workspace

## 📖 Additional Resources

- [Bolt for Go Documentation](../README.md)
- [Slack API Documentation](https://api.slack.com/)
- [Block Kit Builder](https://app.slack.com/block-kit-builder)
- [Slack App Manifest](https://api.slack.com/reference/manifests)

## 🆘 Getting Help

- Check the [main README](../README.md) for framework documentation
- Review [test files](../test/) for implementation examples
- Open an issue for bugs or feature requests
- Join the Slack community for discussions
