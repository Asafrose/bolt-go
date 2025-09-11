# Getting Started with Bolt for Go

This example demonstrates the basic usage of the Bolt framework for Go, equivalent to the getting-started-typescript example from the JavaScript version.

## Features Demonstrated

- Basic app setup with token and signing secret
- Global middleware usage
- Message listeners with pattern matching
- Block-based responses with interactive elements
- Action listeners for button clicks
- Environment variable configuration

## Setup

1. Set your environment variables:
   ```bash
   export SLACK_BOT_TOKEN=xoxb-your-bot-token
   export SLACK_SIGNING_SECRET=your-signing-secret
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

## Usage

- Send a message containing "hello" in any channel where your bot is present
- The bot will respond with a greeting and a button
- Click the button to see the action handler in action

## Key Differences from JavaScript

- Uses Go's explicit error handling instead of async/await
- Context-based parameter passing instead of destructured parameters
- Strong typing for all Slack API objects
- Environment variable handling using `os.Getenv()`
