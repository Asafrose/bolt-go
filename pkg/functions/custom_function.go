package functions

import (
	"errors"
	"fmt"

	"github.com/Asafrose/bolt-go/pkg/middleware"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
)

// CustomFunctionOptions represents options for custom functions
type CustomFunctionOptions struct {
	AutoAcknowledge bool `json:"auto_acknowledge"`
}

// CustomFunction represents a custom function with middleware chain
type CustomFunction struct {
	CallbackID string
	listeners  []types.Middleware[types.SlackCustomFunctionMiddlewareArgs]
	options    CustomFunctionOptions
}

// NewCustomFunctionWithMiddleware creates a new CustomFunction with middleware and options
func NewCustomFunctionWithMiddleware(
	callbackID string,
	middlewares []types.Middleware[types.SlackCustomFunctionMiddlewareArgs],
	options CustomFunctionOptions,
) *CustomFunction {
	// Validate inputs
	validate(callbackID, middlewares)

	return &CustomFunction{
		CallbackID: callbackID,
		listeners:  middlewares,
		options:    options,
	}
}

// GetListeners returns the ordered array of listeners for this custom function
func (cf *CustomFunction) GetListeners() []types.Middleware[types.AllMiddlewareArgs] {
	var listeners []types.Middleware[types.AllMiddlewareArgs]

	// Add built-in middleware in order: onlyEvents, matchEventType, matchCallbackId
	listeners = append(listeners, middleware.OnlyEvents)
	listeners = append(listeners, middleware.MatchEventType("function_executed"))
	listeners = append(listeners, middleware.MatchCallbackId(cf.CallbackID))

	// Add autoAcknowledge if enabled
	if cf.options.AutoAcknowledge {
		listeners = append(listeners, middleware.AutoAcknowledge)
	}

	// Add custom middleware, converting from SlackCustomFunctionMiddlewareArgs to AllMiddlewareArgs
	for _, customMiddleware := range cf.listeners {
		wrappedMiddleware := wrapCustomFunctionMiddleware(customMiddleware)
		listeners = append(listeners, wrappedMiddleware)
	}

	return listeners
}

// wrapCustomFunctionMiddleware wraps a SlackCustomFunctionMiddlewareArgs middleware to work with AllMiddlewareArgs
func wrapCustomFunctionMiddleware(m types.Middleware[types.SlackCustomFunctionMiddlewareArgs]) types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		// Extract or create custom function args from context
		var customFunctionArgs types.SlackCustomFunctionMiddlewareArgs

		// Try to get existing middleware args from context
		if args.Context != nil && args.Context.Custom != nil {
			if middlewareArgs, exists := args.Context.Custom["middlewareArgs"]; exists {
				if existingArgs, ok := middlewareArgs.(types.SlackCustomFunctionMiddlewareArgs); ok {
					customFunctionArgs = existingArgs
				}
			}
		}

		// Ensure we have the basic structure
		if customFunctionArgs.Context == nil {
			customFunctionArgs.AllMiddlewareArgs = args
		}

		// Create wrapped Next function
		customFunctionArgs.Next = args.Next

		// Call the custom middleware
		return m(customFunctionArgs)
	}
}

// validate validates the CustomFunction parameters
func validate(callbackID string, middlewares []types.Middleware[types.SlackCustomFunctionMiddlewareArgs]) {
	// Check callback ID
	if callbackID == "" {
		panic(errors.New("CustomFunction expects a callback_id as the first argument"))
	}

	// Check middleware
	if middlewares == nil {
		panic(errors.New("CustomFunction expects a function or array of functions as the second argument"))
	}

	if len(middlewares) == 0 {
		panic(errors.New("CustomFunction expects at least one middleware function"))
	}

	// In Go, we can't easily validate that all elements are functions at runtime
	// since the type system ensures this at compile time
	// But we can check for nil functions
	for i, middleware := range middlewares {
		if middleware == nil {
			panic(fmt.Errorf("All CustomFunction middleware must be functions, got nil at index %d", i))
		}
	}
}

// CreateFunctionComplete creates a function completion handler
func CreateFunctionComplete(context map[string]interface{}, client *slack.Client) types.FunctionCompleteFn {
	// Validate that we have function_execution_id
	functionExecutionID, exists := context["function_execution_id"]
	if !exists {
		panic(errors.New("function_execution_id is required in context to create complete function"))
	}

	functionExecutionIDStr, ok := functionExecutionID.(string)
	if !ok {
		panic(errors.New("function_execution_id must be a string"))
	}

	return func(outputs map[string]interface{}) error {
		// If outputs is nil, use empty map
		if outputs == nil {
			outputs = make(map[string]interface{})
		}

		// Convert map[string]interface{} to map[string]string for the API
		stringOutputs := make(map[string]string)
		for key, value := range outputs {
			if strValue, ok := value.(string); ok {
				stringOutputs[key] = strValue
			} else {
				stringOutputs[key] = fmt.Sprintf("%v", value)
			}
		}

		// Call Slack API to complete the function
		return client.FunctionCompleteSuccess(functionExecutionIDStr, slack.FunctionCompleteSuccessRequestOptionOutput(stringOutputs))
	}
}

// CreateFunctionFail creates a function failure handler
func CreateFunctionFail(context map[string]interface{}, client *slack.Client) types.FunctionFailFn {
	// Validate that we have function_execution_id
	functionExecutionID, exists := context["function_execution_id"]
	if !exists {
		panic(errors.New("function_execution_id is required in context to create fail function"))
	}

	functionExecutionIDStr, ok := functionExecutionID.(string)
	if !ok {
		panic(errors.New("function_execution_id must be a string"))
	}

	return func(errorMsg string) error {
		// Call Slack API to fail the function
		return client.FunctionCompleteError(functionExecutionIDStr, errorMsg)
	}
}
