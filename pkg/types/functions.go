package types

// CustomFunctionOptions represents options for custom functions
type CustomFunctionOptions struct {
	AutoAcknowledge bool `json:"auto_acknowledge"`
}

// SlackCustomFunctionMiddlewareArgs represents arguments for custom function middleware
type SlackCustomFunctionMiddlewareArgs struct {
	AllMiddlewareArgs
	Event    interface{}        `json:"event"`
	Body     interface{}        `json:"body"`
	Payload  interface{}        `json:"payload"`
	Ack      AckFn[interface{}] `json:"-"`
	Complete FunctionCompleteFn `json:"-"`
	Fail     FunctionFailFn     `json:"-"`
}

// FunctionCompleteFn represents a function to complete a custom function successfully
type FunctionCompleteFn func(outputs map[string]interface{}) error

// FunctionFailFn represents a function to fail a custom function
type FunctionFailFn func(error string) error
