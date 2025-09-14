# Bolt for Go

A **vibe-coded** Go port of the official [Slack Bolt framework](https://github.com/slackapi/bolt-js) for building Slack apps. This implementation follows a Test-Driven Development approach to ensure complete functional parity with the original JavaScript framework.

## Features

This Go port includes all the major features of the JavaScript Bolt framework:

### ✅ Core Features
- **App Configuration**: Initialize apps with tokens, signing secrets, and various options
- **Event Handling**: Listen to Slack events, messages, actions, commands, shortcuts, and more
- **Middleware System**: Global and listener-specific middleware support
- **Multiple Receivers**: HTTP and Socket Mode receivers
- **Authorization**: Flexible authorization system for multi-workspace apps
- **Error Handling**: Comprehensive error types and handling
- **Logging**: Built-in logging with configurable levels

### ✅ Advanced Features  
- **Assistant Support**: Full AI assistant functionality with thread context management
- **Workflow Steps**: Support for legacy workflow steps (deprecated but functional)
- **Custom Functions**: Support for Slack's custom functions
- **Conversation Store**: Built-in conversation context management
- **OAuth Support**: Multi-workspace OAuth installation flow
- **Developer Mode**: Enhanced development experience

### 🔧 Architecture
- **Type Safety**: Strongly typed interfaces throughout
- **Go Idioms**: Follows Go conventions and best practices
- **Concurrent Safe**: Thread-safe operations where needed
- **Extensible**: Easy to extend with custom middleware and receivers

## Vibe-Coded Development Approach

This project is a **vibe-coded port** that follows a rigorous **Test-Driven Development (TDD)** approach to ensure complete functional parity with the original JavaScript Bolt framework:

### Our Vibe-Coding Process

1. **Deep Analysis of JS Implementation**: 
   - Study the original `bolt-js/` codebase to understand behavior and architecture
   - Extract test cases from the comprehensive `bolt-js/test/` directory
   - Map JavaScript functionality to equivalent Go patterns

2. **Test-First Implementation**:
   - **Write Go Tests First**: Create equivalent Go tests that match JS behavior exactly
   - **Maintain Test Parity**: Every JS test case gets a corresponding Go implementation
   - **Verify Behavior**: Tests must pass with identical behavior to the JS version

3. **Implementation with Go Idioms**:
   - **Go-Native Code**: Write idiomatic Go code while maintaining API compatibility
   - **Type Safety**: Leverage Go's strong typing for better developer experience
   - **Error Handling**: Use explicit Go error handling patterns instead of JS exceptions

4. **Continuous Parity Verification**:
   - **Automated Analysis**: Use scripts to track test coverage and parity
   - **Behavior Matching**: Ensure Go implementation produces identical results to JS
   - **Documentation**: Maintain detailed mapping in `TEST_PARITY_ANALYSIS.md`

### Test Suite Matching Process

Our comprehensive test matching ensures that every aspect of the JavaScript Bolt framework is accurately ported:

- **1:1 Test Mapping**: Each JavaScript test case has a corresponding Go test
- **Behavioral Equivalence**: Tests verify identical behavior, not just API compatibility
- **Edge Case Coverage**: All JavaScript edge cases and error conditions are replicated
- **Integration Testing**: Full workflow testing to ensure end-to-end compatibility

**Current Achievement**: 97.3% test parity (364 of 374 JS tests implemented)

This approach ensures that developers familiar with the JavaScript Bolt framework can seamlessly transition to the Go version with confidence that all functionality works identically.

## Installation

```bash
go get github.com/Asafrose/bolt-go
```

## Quick Start

### Basic App

```go
package main

import (
    "context"
    "log"
    "os"
    
    bolt "github.com/Asafrose/bolt-go"
    "github.com/Asafrose/bolt-go/pkg/types"
)

func main() {
    // Initialize the app
    app, err := bolt.New(bolt.AppOptions{
        Token:         os.Getenv("SLACK_BOT_TOKEN"),
        SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
        LogLevel:      bolt.LogLevelDebug,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Listen for app mentions using typed event constants
    app.Event(types.EventTypeAppMention, func(args types.SlackEventMiddlewareArgs) error {
        // Respond to the mention
        if args.Say != nil {
            _, err := args.Say(&types.SayArguments{
                Text: "Hello! 👋",
            })
            return err
        }
        return nil
    })

    // Listen for slash commands
    app.Command("/hello", func(args types.SlackCommandMiddlewareArgs) error {
        return args.Ack(&types.CommandResponse{
            Text: "Hello from Go! 🚀",
        })
    })

    // Start the app
    ctx := context.Background()
    log.Println("⚡️ Bolt app is running!")
    if err := app.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Socket Mode App

```go
app, err := bolt.New(bolt.AppOptions{
    Token:      os.Getenv("SLACK_BOT_TOKEN"),
    AppToken:   os.Getenv("SLACK_APP_TOKEN"),
    SocketMode: true,
    LogLevel:   bolt.LogLevelDebug,
})
```

### Multi-Workspace App

```go
app, err := bolt.New(bolt.AppOptions{
    SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
    ClientID:      os.Getenv("SLACK_CLIENT_ID"),
    ClientSecret:  os.Getenv("SLACK_CLIENT_SECRET"),
    Authorize: func(ctx context.Context, source bolt.AuthorizeSourceData, body interface{}) (*bolt.AuthorizeResult, error) {
        // Your authorization logic here
        return &bolt.AuthorizeResult{
            BotToken: "token-for-workspace",
        }, nil
    },
})
```

## API Documentation

### App Methods

```go
// Event listeners using typed constants (recommended)
app.Event(types.EventTypeAppMention, middleware...)
app.Event(types.EventTypeMessage, middleware...)

// Message listeners with pattern matching
app.Message("hello", middleware...)           // String matching
app.Message(regexp.MustCompile(`hi.*`), middleware...) // Regex matching

// Action listeners  
app.Action(types.ActionConstraints{ActionID: "button_id"}, middleware...)
app.Action(types.ActionConstraints{BlockID: "block_id"}, middleware...)

// Command listeners
app.Command("/command", middleware...)
app.Command(regexp.MustCompile(`/test.*`), middleware...) // Regex support

// Shortcut listeners
app.Shortcut(types.ShortcutConstraints{CallbackID: "shortcut_id"}, middleware...)
app.Shortcut(types.ShortcutConstraints{Type: "global"}, middleware...)

// View listeners
app.View(types.ViewConstraints{CallbackID: "view_id"}, middleware...)
app.View(types.ViewConstraints{Type: "view_submission"}, middleware...)

// Options listeners
app.Options(types.OptionsConstraints{ActionID: "select_id"}, middleware...)

// Custom Function listeners
app.Function("callback_id", middleware...)

// Global middleware
app.Use(middleware...)
```

### Assistant Support

```go
// Create an assistant with thread context management
assistant, err := bolt.NewAssistant(bolt.AssistantConfig{
    ThreadContextStore: bolt.NewDefaultThreadContextStore(),
    ThreadStarted: []bolt.AssistantThreadStartedMiddleware{
        func(args bolt.AssistantThreadStartedMiddlewareArgs) error {
            _, err := args.Say(&types.SayArguments{
                Text: "Hello! I'm your AI assistant. How can I help?",
            })
            return err
        },
    },
    UserMessage: []bolt.AssistantUserMessageMiddleware{
        func(args bolt.AssistantUserMessageMiddlewareArgs) error {
            // Process user message and respond
            _, err := args.Say(&types.SayArguments{
                Text: "I understand you said: " + args.Message.Text,
            })
            return err
        },
    },
    ThreadContextChanged: []bolt.AssistantThreadContextChangedMiddleware{
        func(args bolt.AssistantThreadContextChangedMiddlewareArgs) error {
            args.Logger.Info("Thread context changed", "context", args.Context)
            return nil
        },
    },
})

// Add assistant middleware to your app
app.Use(assistant.GetMiddleware())
```

### Custom Functions Support

```go
// Register a custom function handler
app.Function("my_function_callback_id", func(args types.SlackCustomFunctionMiddlewareArgs) error {
    // Process the function execution
    inputs := args.Inputs // Function inputs from Slack
    
    // Perform your custom logic here
    result := processInputs(inputs)
    
    // Complete the function successfully
    return args.Complete(map[string]interface{}{
        "result": result,
    })
})

// Handle function failures
app.Function("another_function", func(args types.SlackCustomFunctionMiddlewareArgs) error {
    if err := validateInputs(args.Inputs); err != nil {
        // Fail the function with an error message
        return args.Fail("Invalid inputs provided")
    }
    
    // Continue with processing...
    return args.Complete(map[string]interface{}{"status": "success"})
})
```

### Conversation Store

```go
// Use built-in memory store for conversation context
type MyConversationState struct {
    UserPreference string `json:"user_preference"`
    LastAction     string `json:"last_action"`
}

// The conversation store is automatically initialized
// Access conversation context in middleware
app.Event(types.EventTypeMessage, func(args types.SlackEventMiddlewareArgs) error {
    // Conversation context is available in args.Context.ConversationContext
    if args.Context.ConversationContext != nil {
        // Use existing context
        state := args.Context.ConversationContext.(*MyConversationState)
        args.Logger.Info("Previous state", "preference", state.UserPreference)
    }
    
    // Update conversation state
    newState := &MyConversationState{
        UserPreference: "dark_mode",
        LastAction:     "message_received",
    }
    
    // Store will be automatically saved
    args.Context.ConversationContext = newState
    return nil
})
```

### Workflow Steps (Deprecated)

```go
// Create a workflow step (legacy support)
workflowStep := bolt.NewWorkflowStep("my_workflow_step", bolt.WorkflowStepConfig{
    Edit: func(args bolt.WorkflowStepEditMiddlewareArgs) error {
        // Configure the step
        return args.Configure(slack.WorkflowStepConfiguration{
            // Step configuration
        })
    },
    Save: func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
        // Save step configuration
        return args.Update(map[string]interface{}{
            "configured": true,
        })
    },
    Execute: func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
        // Execute the step
        return args.Complete(map[string]interface{}{
            "result": "Step completed successfully",
        })
    },
})

// Add workflow step to app
app.Use(workflowStep.GetMiddleware())
```

### Error Handling

```go
// Custom global error handler
app.Error(func(err error) {
    if codedErr, ok := bolt.AsCodedError(err); ok {
        log.Printf("Coded error [%s]: %v", codedErr.Code(), codedErr)
    } else {
        log.Printf("Uncoded error: %v", err)
    }
})

// Middleware with error handling
app.Use(func(args types.AllMiddlewareArgs) error {
    defer func() {
        if r := recover(); r != nil {
            args.Logger.Error("Panic recovered", "panic", r)
        }
    }()
    return args.Next()
})

// Authorization error handling
app.Use(func(args types.AllMiddlewareArgs) error {
    // Custom authorization logic
    if !isAuthorized(args.Context) {
        return bolt.NewAuthorizationError("User not authorized")
    }
    return args.Next()
})
```

## Testing

The project includes comprehensive tests that mirror the JavaScript Bolt framework tests:

```bash
# Run all tests
go test ./test/... -v

# Run specific test suite
go test ./test -run TestApp -v

# Run tests with coverage
go test ./test/... -cover

# Compare test coverage with JavaScript version
go run scripts/compare_tests.go

# Generate comprehensive parity analysis
go run scripts/comprehensive_analysis.go
```

### Test Coverage

Current test coverage compared to bolt-js:
- **JavaScript**: 374 test cases across 31 test files
- **Go**: 776 test cases across 47 test files
- **Coverage**: 97.3% (364 of 374 JS tests implemented)

#### Coverage by Module:
- **Assistant**: 100% (32/32 tests)
- **AssistantThreadContextStore**: 100% (8/8 tests)
- **AwsLambdaReceiver**: 100% (14/14 tests)
- **CustomFunction**: 100% (11/11 tests)
- **ExpressReceiver**: 77.3% (34/44 tests) - Node.js specific features marked N/A
- **HTTPModuleFunctions**: 100% (19/19 tests)
- **SocketModeReceiver**: 100% (26/26 tests)
- **Core App**: 100% (26/26 tests)
- **Routing**: 100% across all routing types
- **Middleware**: 100% (27/27 tests)
- **Built-in Middleware**: 100% (27/27 tests)
- **Conversation Store**: 100% (8/8 tests)
- **Error Handling**: 100% (3/3 tests)
- **Helpers**: 100% (11/11 tests)
- **Request Verification**: 100% (6/6 tests)

See `TEST_PARITY_ANALYSIS.md` for detailed test mapping.

## Project Structure

```
bolt-go/
├── bolt.go              # Main package exports and type aliases
├── pkg/
│   ├── app/            # Core App implementation
│   ├── assistant/      # AI Assistant functionality with thread context
│   ├── conversation/   # Conversation store and middleware
│   ├── errors/         # Comprehensive error types and handling
│   ├── functions/      # Custom Functions support
│   ├── helpers/        # Utility functions and parsers
│   ├── http/           # HTTP utilities and request handling
│   ├── middleware/     # Built-in middleware functions
│   ├── oauth/          # OAuth installation and state management
│   ├── receivers/      # HTTP, Socket Mode, and AWS Lambda receivers
│   ├── types/          # Type definitions for Slack API objects
│   └── workflow/       # Workflow step functionality (deprecated)
├── test/               # Comprehensive test suites (47 files, 754 tests)
├── examples/           # Working examples for different use cases
├── scripts/            # Development and analysis scripts
└── bolt-js/            # Original JavaScript implementation (reference)
```

## Compatibility & Go Adaptations

This Go port maintains functional compatibility with the JavaScript Bolt framework while following Go conventions:

### Language Adaptations
- **Type Safety**: Strongly typed interfaces with compile-time safety
- **Error Handling**: Explicit error returns instead of exceptions
- **Concurrency**: Go routines and channels for concurrent operations
- **Memory Management**: Automatic garbage collection, no manual memory management
- **Package System**: Go modules instead of npm packages

### API Adaptations
- **Configuration**: Struct-based options with clear field types
- **Middleware**: Function-based middleware with explicit error handling
- **Callbacks**: Interface-based patterns instead of callback functions
- **Async Operations**: Channel-based or direct return patterns
- **Event Constants**: Typed constants for event types (e.g., `types.EventTypeAppMention`)

### Performance Benefits
- **Compiled Binary**: No runtime interpretation overhead
- **Static Linking**: Single binary deployment
- **Memory Efficiency**: Lower memory footprint than Node.js
- **Concurrent Processing**: Native goroutine support for handling multiple requests

## Current Implementation Status

This vibe-coded Go port has achieved **97.3% feature parity** with the JavaScript Bolt framework through comprehensive test matching:

### ✅ Fully Implemented (100% test parity)
1. **Core App functionality** - Complete with all routing methods (26/26 tests)
2. **HTTP Receiver & Module Functions** - Full request handling and verification (19/19 tests)
3. **Socket Mode Receiver** - Real-time event processing (26/26 tests)
4. **AWS Lambda Receiver** - Serverless deployment support (14/14 tests)
5. **Assistant Support** - AI assistant with thread context management (32/32 tests)
6. **Assistant Thread Context Store** - Thread context management (8/8 tests)
7. **Custom Functions** - Slack's custom function framework (11/11 tests)
8. **Conversation Store** - Built-in conversation context management (8/8 tests)
9. **Middleware System** - Global and listener-specific middleware (27/27 tests)
10. **Built-in Middleware** - All built-in middleware functions (27/27 tests)
11. **Error Handling** - Comprehensive error types and handling (3/3 tests)
12. **Request Verification** - Complete signature and timestamp verification (6/6 tests)
13. **Helpers & Utilities** - All utility functions (11/11 tests)
14. **All Routing Types** - Events, commands, actions, shortcuts, views, options, messages
15. **Type Safety** - Strongly typed interfaces throughout

### 🟡 Partially Implemented (Node.js specific features)
1. **Express Receiver** - 77.3% (34/44 tests) - Node.js specific features marked as N/A
2. **Workflow Steps** - Legacy support (deprecated in Slack, but functional)

### 🔴 Not Applicable to Go
1. **Node.js Express middleware** - Replaced with Go HTTP patterns
2. **Node.js specific body parsing** - Replaced with Go equivalents
3. **Express server lifecycle** - Replaced with Go HTTP server patterns

### Test Coverage: 97.3%
- **776 Go test cases** covering **364 of 374 JavaScript test scenarios**
- **Comprehensive test suites** for all major components with near-complete parity
- **Integration tests** for real-world usage patterns
- **Continuous parity tracking** with automated analysis scripts
- **Behavioral equivalence** verified through extensive test matching

## License

MIT License - see the original [bolt-js repository](https://github.com/slackapi/bolt-js) for details.

## Development & Contributing

### Development Guidelines

For developers contributing to this project:

### Parity Tracking
Use the analysis scripts to track implementation progress:

```bash
# Generate comprehensive parity analysis
go run scripts/comprehensive_analysis.go

# Compare test coverage
go run scripts/compare_tests.go

# Update parity documentation
go run scripts/update_analysis.go
```

### Contributing Guidelines
1. **Reference JS Implementation**: Always check `bolt-js/` for expected behavior
2. **Write Tests First**: Implement corresponding Go tests before features
3. **Update Documentation**: Keep `TEST_PARITY_ANALYSIS.md` current
4. **Follow Go Conventions**: Use Go idioms while maintaining API compatibility
5. **Test Coverage**: Aim for equivalent test coverage to JavaScript version

## Acknowledgments

This is a port of the official Slack Bolt framework for JavaScript/TypeScript. All credit for the original design and architecture goes to the Slack team and contributors of the [bolt-js project](https://github.com/slackapi/bolt-js).

The Go port maintains the same MIT license and aims to provide the same developer experience while leveraging Go's performance and type safety benefits.
