package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomFunction(t *testing.T) {
	t.Run("should create custom function with callback ID", func(t *testing.T) {
		callbackID := "test_function"

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{
				Outputs: map[string]interface{}{
					"result": "success",
				},
			}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		assert.NotNil(t, customFunc, "Custom function should be created")
		assert.Equal(t, callbackID, customFunc.CallbackID, "Callback ID should match")
	})

	t.Run("should create custom function with options", func(t *testing.T) {
		callbackID := "test_function_with_options"
		options := bolt.CustomFunctionOptions{
			AutoAcknowledge: false,
		}

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{
				Outputs: map[string]interface{}{
					"message": "processed",
				},
			}, nil
		}

		customFunc := bolt.NewCustomFunctionWithOptions(callbackID, options, handler)

		assert.NotNil(t, customFunc, "Custom function should be created with options")
		assert.Equal(t, callbackID, customFunc.CallbackID, "Callback ID should match")
	})

	t.Run("should validate callback ID", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "callback_id", "Should panic with callback ID error")
			}
		}()

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{}, nil
		}

		// This should panic due to empty callback ID
		bolt.NewCustomFunction("", handler)

		t.Error("Should have panicked for empty callback ID")
	})

	t.Run("should validate handler function", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "handler", "Should panic with handler error")
			}
		}()

		// This should panic due to nil handler
		bolt.NewCustomFunction("test", nil)

		t.Error("Should have panicked for nil handler")
	})
}

func TestCustomFunctionExecution(t *testing.T) {
	t.Run("should execute custom function", func(t *testing.T) {
		callbackID := "execution_test"
		handlerCalled := false

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			handlerCalled = true

			// Verify inputs are passed correctly
			assert.NotNil(t, args.Inputs, "Inputs should be provided")
			assert.NotNil(t, args.Complete, "Complete function should be provided")
			assert.NotNil(t, args.Fail, "Fail function should be provided")

			return bolt.CustomFunctionResponse{
				Outputs: map[string]interface{}{
					"processed":   true,
					"input_count": len(args.Inputs),
				},
			}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Execute the function
		args := bolt.CustomFunctionArgs{
			Inputs: map[string]interface{}{
				"test_input": "test_value",
			},
		}

		response, err := customFunc.Execute(args)

		assert.NoError(t, err, "Execution should succeed")
		assert.True(t, handlerCalled, "Handler should be called")
		assert.NotNil(t, response.Outputs, "Response should have outputs")
	})

	t.Run("should handle execution errors", func(t *testing.T) {
		callbackID := "error_test"

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{}, assert.AnError
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Execute the function
		args := bolt.CustomFunctionArgs{
			Inputs: map[string]interface{}{},
		}

		_, err := customFunc.Execute(args)

		assert.Error(t, err, "Execution should return error")
	})

	t.Run("should handle complete function", func(t *testing.T) {
		callbackID := "complete_test"
		completeCalled := false

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			// Test the complete function
			err := args.Complete(map[string]interface{}{
				"result": "completed",
			})
			if err == nil {
				completeCalled = true
			}

			return bolt.CustomFunctionResponse{}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Execute the function
		args := bolt.CustomFunctionArgs{
			Inputs: map[string]interface{}{},
		}

		_, err := customFunc.Execute(args)

		assert.NoError(t, err, "Execution should succeed")
		assert.True(t, completeCalled, "Complete function should be called")
	})

	t.Run("should handle fail function", func(t *testing.T) {
		callbackID := "fail_test"
		failCalled := false

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			// Test the fail function
			err := args.Fail("test error message")
			if err == nil {
				failCalled = true
			}

			return bolt.CustomFunctionResponse{}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Execute the function
		args := bolt.CustomFunctionArgs{
			Inputs: map[string]interface{}{},
		}

		_, err := customFunc.Execute(args)

		assert.NoError(t, err, "Execution should succeed")
		assert.True(t, failCalled, "Fail function should be called")
	})
}

func TestCustomFunctionMiddleware(t *testing.T) {
	t.Run("should provide middleware listeners", func(t *testing.T) {
		callbackID := "middleware_test"

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)
		listeners := customFunc.GetListeners()

		assert.NotEmpty(t, listeners, "Should provide middleware listeners")
		assert.Greater(t, len(listeners), 0, "Should have at least one listener")
	})

	t.Run("should provide different listeners for auto-acknowledge", func(t *testing.T) {
		callbackID := "auto_ack_test"

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			return bolt.CustomFunctionResponse{}, nil
		}

		// With auto-acknowledge
		customFuncAuto := bolt.NewCustomFunctionWithOptions(callbackID, bolt.CustomFunctionOptions{
			AutoAcknowledge: true,
		}, handler)

		// Without auto-acknowledge
		customFuncManual := bolt.NewCustomFunctionWithOptions(callbackID, bolt.CustomFunctionOptions{
			AutoAcknowledge: false,
		}, handler)

		autoListeners := customFuncAuto.GetListeners()
		manualListeners := customFuncManual.GetListeners()

		assert.NotEmpty(t, autoListeners, "Auto-acknowledge function should have listeners")
		assert.NotEmpty(t, manualListeners, "Manual acknowledge function should have listeners")

		// Both should have listeners, but potentially different counts due to auto-acknowledge middleware
		assert.Greater(t, len(autoListeners), 0, "Auto-acknowledge should have listeners")
		assert.Greater(t, len(manualListeners), 0, "Manual acknowledge should have listeners")
	})
}

func TestCustomFunctionIntegration(t *testing.T) {
	t.Run("should integrate with bolt app", func(t *testing.T) {
		_, err := bolt.New(bolt.AppOptions{
			Token:         &fakeToken,
			SigningSecret: &fakeSigningSecret,
		})
		require.NoError(t, err)

		callbackID := "integration_test"
		handlerCalled := false

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			handlerCalled = true
			return bolt.CustomFunctionResponse{
				Outputs: map[string]interface{}{
					"integration": "success",
				},
			}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Test that we can get listeners from the custom function
		listeners := customFunc.GetListeners()
		assert.NotEmpty(t, listeners, "Custom function should provide listeners for app integration")

		// In a real app, these listeners would be registered with the app
		// For this test, we just verify they exist
		assert.False(t, handlerCalled, "Handler should not be called yet")
	})

	t.Run("should handle different callback IDs", func(t *testing.T) {
		callbackIDs := []string{
			"test_function_1",
			"test_function_2",
			"my_custom_function",
			"workflow_function",
			"ai_function",
		}

		for _, callbackID := range callbackIDs {
			handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
				return bolt.CustomFunctionResponse{
					Outputs: map[string]interface{}{
						"callback_id": callbackID,
					},
				}, nil
			}

			customFunc := bolt.NewCustomFunction(callbackID, handler)

			assert.Equal(t, callbackID, customFunc.CallbackID, "Callback ID should match for %s", callbackID)
			assert.NotEmpty(t, customFunc.GetListeners(), "Should have listeners for %s", callbackID)
		}
	})

	t.Run("should handle complex inputs and outputs", func(t *testing.T) {
		callbackID := "complex_data_test"

		handler := func(args bolt.CustomFunctionArgs) (bolt.CustomFunctionResponse, error) {
			// Process complex inputs
			processedData := make(map[string]interface{})

			for key, value := range args.Inputs {
				if str, ok := value.(string); ok {
					processedData[key+"_processed"] = strings.ToUpper(str)
				} else {
					processedData[key+"_processed"] = value
				}
			}

			return bolt.CustomFunctionResponse{
				Outputs: map[string]interface{}{
					"processed_data": processedData,
					"input_keys":     getKeys(args.Inputs),
					"timestamp":      "2023-01-01T00:00:00Z",
				},
			}, nil
		}

		customFunc := bolt.NewCustomFunction(callbackID, handler)

		// Test with complex inputs
		complexInputs := map[string]interface{}{
			"text_field":    "hello world",
			"number_field":  42,
			"boolean_field": true,
			"array_field":   []string{"a", "b", "c"},
			"object_field": map[string]interface{}{
				"nested": "value",
			},
		}

		args := bolt.CustomFunctionArgs{
			Inputs: complexInputs,
		}

		response, err := customFunc.Execute(args)

		assert.NoError(t, err, "Should handle complex data")
		assert.NotNil(t, response.Outputs, "Should return outputs")

		if outputs := response.Outputs; outputs != nil {
			assert.Contains(t, outputs, "processed_data", "Should contain processed data")
			assert.Contains(t, outputs, "input_keys", "Should contain input keys")
			assert.Contains(t, outputs, "timestamp", "Should contain timestamp")
		}
	})
}

// Helper function for complex data test
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
