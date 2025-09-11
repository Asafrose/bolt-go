# AWS Lambda Deployment Example

This example demonstrates how to deploy a Bolt for Go app to AWS Lambda using the Serverless Framework.

## Features Demonstrated

- AWS Lambda receiver setup
- Message listeners with pattern matching
- Button action handlers
- Serverless Framework configuration
- Cross-compilation for Linux deployment

## Prerequisites

- [Serverless Framework](https://www.serverless.com/) installed globally
- AWS CLI configured with appropriate credentials
- Go 1.21 or later

## Setup

1. Install the Serverless Framework:
   ```bash
   npm install -g serverless
   ```

2. Set your environment variables:
   ```bash
   export SLACK_BOT_TOKEN=xoxb-your-bot-token
   export SLACK_SIGNING_SECRET=your-signing-secret
   ```

3. Install dependencies:
   ```bash
   go mod tidy
   ```

## Deployment

1. Build the binary for Linux:
   ```bash
   make build
   ```

2. Deploy to AWS Lambda:
   ```bash
   make deploy
   ```

3. The deployment will output an API Gateway endpoint URL. Use this URL as your Slack app's Request URL.

## Configuration

Update your Slack app configuration:
- Set the Request URL to the API Gateway endpoint + `/slack/events`
- Example: `https://your-api-id.execute-api.us-east-1.amazonaws.com/dev/slack/events`

## Usage

- Send "hello" to trigger the hello message handler
- Send "goodbye" to trigger the goodbye message handler
- Click buttons to test action handlers

## Commands

- `make build` - Build the binary for Linux
- `make deploy` - Deploy to AWS Lambda
- `make remove` - Remove the deployment
- `make logs` - View Lambda function logs

## Key Differences from HTTP Receiver

- Uses AWS Lambda receiver instead of HTTP receiver
- Initializes the app in the `init()` function for Lambda cold starts
- Uses the `lambda.Start()` function to run the handler
- No need to specify a port (handled by API Gateway)
- Cross-compilation required for Linux deployment

## Cost Considerations

AWS Lambda pricing is based on:
- Number of requests
- Duration of execution
- Memory allocation

For most Slack apps, Lambda's free tier should cover development and small-scale usage.
