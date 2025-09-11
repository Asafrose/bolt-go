package test

import (
	"testing"

	"github.com/Asafrose/bolt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowStep(t *testing.T) {
	t.Run("constructor", func(t *testing.T) {
		t.Run("should create workflow step with required middleware", func(t *testing.T) {
			config := bolt.WorkflowStepConfig{
				Edit: []bolt.WorkflowStepEditMiddleware{
					func(args bolt.WorkflowStepEditMiddlewareArgs) error {
						return args.Next()
					},
				},
				Save: []bolt.WorkflowStepSaveMiddleware{
					func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
						return args.Next()
					},
				},
				Execute: []bolt.WorkflowStepExecuteMiddleware{
					func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			step, err := bolt.NewWorkflowStep("test_step", config)
			require.NoError(t, err)
			assert.NotNil(t, step)
		})

		t.Run("should fail without edit middleware", func(t *testing.T) {
			config := bolt.WorkflowStepConfig{
				Save: []bolt.WorkflowStepSaveMiddleware{
					func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
						return args.Next()
					},
				},
				Execute: []bolt.WorkflowStepExecuteMiddleware{
					func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			_, err := bolt.NewWorkflowStep("test_step", config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "edit middleware is required")
		})

		t.Run("should fail without save middleware", func(t *testing.T) {
			config := bolt.WorkflowStepConfig{
				Edit: []bolt.WorkflowStepEditMiddleware{
					func(args bolt.WorkflowStepEditMiddlewareArgs) error {
						return args.Next()
					},
				},
				Execute: []bolt.WorkflowStepExecuteMiddleware{
					func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			_, err := bolt.NewWorkflowStep("test_step", config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "save middleware is required")
		})

		t.Run("should fail without execute middleware", func(t *testing.T) {
			config := bolt.WorkflowStepConfig{
				Edit: []bolt.WorkflowStepEditMiddleware{
					func(args bolt.WorkflowStepEditMiddlewareArgs) error {
						return args.Next()
					},
				},
				Save: []bolt.WorkflowStepSaveMiddleware{
					func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			_, err := bolt.NewWorkflowStep("test_step", config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "execute middleware is required")
		})
	})

	t.Run("middleware integration", func(t *testing.T) {
		t.Run("should return middleware function", func(t *testing.T) {
			config := bolt.WorkflowStepConfig{
				Edit: []bolt.WorkflowStepEditMiddleware{
					func(args bolt.WorkflowStepEditMiddlewareArgs) error {
						return args.Next()
					},
				},
				Save: []bolt.WorkflowStepSaveMiddleware{
					func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
						return args.Next()
					},
				},
				Execute: []bolt.WorkflowStepExecuteMiddleware{
					func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			step, err := bolt.NewWorkflowStep("test_step", config)
			require.NoError(t, err)

			middleware := step.GetMiddleware()
			assert.NotNil(t, middleware)
		})

		t.Run("should integrate with app", func(t *testing.T) {
			app, err := bolt.New(bolt.AppOptions{
				Token:         &fakeToken,
				SigningSecret: &fakeSigningSecret,
			})
			require.NoError(t, err)

			config := bolt.WorkflowStepConfig{
				Edit: []bolt.WorkflowStepEditMiddleware{
					func(args bolt.WorkflowStepEditMiddlewareArgs) error {
						return args.Next()
					},
				},
				Save: []bolt.WorkflowStepSaveMiddleware{
					func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
						return args.Next()
					},
				},
				Execute: []bolt.WorkflowStepExecuteMiddleware{
					func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
						return args.Next()
					},
				},
			}

			step, err := bolt.NewWorkflowStep("test_step", config)
			require.NoError(t, err)

			// This should not panic
			app.Use(step.GetMiddleware())
		})
	})
}

func TestWorkflowStepUtilities(t *testing.T) {
	t.Run("utility functions", func(t *testing.T) {
		// Test utility function creation and usage
		// This would typically be tested through integration tests
		// with actual workflow step middleware

		config := bolt.WorkflowStepConfig{
			Edit: []bolt.WorkflowStepEditMiddleware{
				func(args bolt.WorkflowStepEditMiddlewareArgs) error {
					// Test that utility functions are available
					assert.NotNil(t, args.Configure)
					assert.NotNil(t, args.Update)
					assert.NotNil(t, args.Complete)
					assert.NotNil(t, args.Fail)
					return args.Next()
				},
			},
			Save: []bolt.WorkflowStepSaveMiddleware{
				func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
					// Test that utility functions are available
					assert.NotNil(t, args.Update)
					assert.NotNil(t, args.Complete)
					assert.NotNil(t, args.Fail)
					return args.Next()
				},
			},
			Execute: []bolt.WorkflowStepExecuteMiddleware{
				func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
					// Test that utility functions are available
					assert.NotNil(t, args.Complete)
					assert.NotNil(t, args.Fail)
					return args.Next()
				},
			},
		}

		step, err := bolt.NewWorkflowStep("test_step", config)
		require.NoError(t, err)
		assert.NotNil(t, step)
	})
}

func TestWorkflowStepDeprecation(t *testing.T) {
	t.Run("should still work despite deprecation", func(t *testing.T) {
		// Even though workflow steps are deprecated, the functionality
		// should still work for backward compatibility

		config := bolt.WorkflowStepConfig{
			Edit: []bolt.WorkflowStepEditMiddleware{
				func(args bolt.WorkflowStepEditMiddlewareArgs) error {
					return args.Next()
				},
			},
			Save: []bolt.WorkflowStepSaveMiddleware{
				func(args bolt.WorkflowStepSaveMiddlewareArgs) error {
					return args.Next()
				},
			},
			Execute: []bolt.WorkflowStepExecuteMiddleware{
				func(args bolt.WorkflowStepExecuteMiddlewareArgs) error {
					return args.Next()
				},
			},
		}

		step, err := bolt.NewWorkflowStep("deprecated_step", config)
		require.NoError(t, err)
		assert.NotNil(t, step)

		middleware := step.GetMiddleware()
		assert.NotNil(t, middleware)
	})
}
