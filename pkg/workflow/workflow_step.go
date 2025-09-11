package workflow

import (
	"encoding/json"

	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
)

// Deprecated: WorkflowStep functionality is deprecated as Steps from Apps are no longer supported

// StepConfigureArguments represents arguments for configuring a workflow step
type StepConfigureArguments struct {
	Blocks          []slack.Block `json:"blocks"`
	PrivateMetadata *string       `json:"private_metadata,omitempty"`
	SubmitDisabled  *bool         `json:"submit_disabled,omitempty"`
	ExternalID      *string       `json:"external_id,omitempty"`
}

// StepUpdateArguments represents arguments for updating a workflow step
type StepUpdateArguments struct {
	Inputs       map[string]StepInput `json:"inputs,omitempty"`
	Outputs      []StepOutput         `json:"outputs,omitempty"`
	StepName     *string              `json:"step_name,omitempty"`
	StepImageURL *string              `json:"step_image_url,omitempty"`
}

// StepInput represents a workflow step input
type StepInput struct {
	Value                   interface{}            `json:"value"`
	SkipVariableReplacement *bool                  `json:"skip_variable_replacement,omitempty"`
	Variables               map[string]interface{} `json:"variables,omitempty"`
}

// StepOutput represents a workflow step output
type StepOutput struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// StepCompleteArguments represents arguments for completing a workflow step
type StepCompleteArguments struct {
	Outputs map[string]interface{} `json:"outputs,omitempty"`
}

// StepFailArguments represents arguments for failing a workflow step
type StepFailArguments struct {
	Error StepError `json:"error"`
}

// StepError represents a workflow step error
type StepError struct {
	Message string `json:"message"`
}

// Function types for workflow step operations
type StepConfigureFn func(args StepConfigureArguments) error
type StepUpdateFn func(args *StepUpdateArguments) error
type StepCompleteFn func(args *StepCompleteArguments) error
type StepFailFn func(args StepFailArguments) error

// Middleware types
type WorkflowStepEditMiddleware func(args WorkflowStepEditMiddlewareArgs) error
type WorkflowStepSaveMiddleware func(args WorkflowStepSaveMiddlewareArgs) error
type WorkflowStepExecuteMiddleware func(args WorkflowStepExecuteMiddlewareArgs) error

// Middleware argument types
type WorkflowStepEditMiddlewareArgs struct {
	types.AllMiddlewareArgs
	Step      interface{}     `json:"step"`
	Body      interface{}     `json:"body"`
	Configure StepConfigureFn `json:"-"`
	Update    StepUpdateFn    `json:"-"`
	Complete  StepCompleteFn  `json:"-"`
	Fail      StepFailFn      `json:"-"`
}

type WorkflowStepSaveMiddlewareArgs struct {
	types.AllMiddlewareArgs
	Step     interface{}    `json:"step"`
	Body     interface{}    `json:"body"`
	View     interface{}    `json:"view"`
	Update   StepUpdateFn   `json:"-"`
	Complete StepCompleteFn `json:"-"`
	Fail     StepFailFn     `json:"-"`
}

type WorkflowStepExecuteMiddlewareArgs struct {
	types.AllMiddlewareArgs
	Step     interface{}    `json:"step"`
	Body     interface{}    `json:"body"`
	Event    interface{}    `json:"event"`
	Complete StepCompleteFn `json:"-"`
	Fail     StepFailFn     `json:"-"`
}

// WorkflowStepConfig represents configuration for a workflow step
type WorkflowStepConfig struct {
	Edit    []WorkflowStepEditMiddleware    `json:"-"`
	Save    []WorkflowStepSaveMiddleware    `json:"-"`
	Execute []WorkflowStepExecuteMiddleware `json:"-"`
}

// WorkflowStep represents a workflow step
type WorkflowStep struct {
	callbackID        string
	editMiddleware    []WorkflowStepEditMiddleware
	saveMiddleware    []WorkflowStepSaveMiddleware
	executeMiddleware []WorkflowStepExecuteMiddleware
}

// NewWorkflowStep creates a new workflow step
// Deprecated: Steps from Apps are no longer supported
func NewWorkflowStep(callbackID string, config WorkflowStepConfig) (*WorkflowStep, error) {
	if len(config.Edit) == 0 {
		return nil, errors.NewWorkflowStepInitializationError("edit middleware is required")
	}

	if len(config.Save) == 0 {
		return nil, errors.NewWorkflowStepInitializationError("save middleware is required")
	}

	if len(config.Execute) == 0 {
		return nil, errors.NewWorkflowStepInitializationError("execute middleware is required")
	}

	return &WorkflowStep{
		callbackID:        callbackID,
		editMiddleware:    config.Edit,
		saveMiddleware:    config.Save,
		executeMiddleware: config.Execute,
	}, nil
}

// GetMiddleware returns the middleware function for the app
func (ws *WorkflowStep) GetMiddleware() types.Middleware[types.AllMiddlewareArgs] {
	return func(args types.AllMiddlewareArgs) error {
		return ws.processEvent(args)
	}
}

// processEvent processes workflow step events
func (ws *WorkflowStep) processEvent(args types.AllMiddlewareArgs) error {
	// This would need access to the event body to determine event type
	eventType := ws.extractEventType(args)

	switch eventType {
	case "workflow_step_edit":
		return ws.processEdit(args)
	case "view_submission":
		if ws.isStepSave(args) {
			return ws.processSave(args)
		}
		return args.Next()
	case "workflow_step_execute":
		return ws.processExecute(args)
	default:
		// Not a workflow step event, continue
		return args.Next()
	}
}

// extractEventType extracts the event type from middleware args
func (ws *WorkflowStep) extractEventType(args types.AllMiddlewareArgs) string {
	// This would need to extract from the actual event body
	// For now, return empty string
	return ""
}

// isStepSave checks if this is a step save event
func (ws *WorkflowStep) isStepSave(args types.AllMiddlewareArgs) bool {
	// This would check if the view submission is for this workflow step
	return false
}

// processEdit processes workflow step edit events
func (ws *WorkflowStep) processEdit(args types.AllMiddlewareArgs) error {
	stepUtilities := ws.createStepUtilities(args)

	middlewareArgs := WorkflowStepEditMiddlewareArgs{
		AllMiddlewareArgs: args,
		Step:              nil, // Would be populated from actual event
		Body:              nil, // Would be populated from actual body
		Configure:         stepUtilities.Configure,
		Update:            stepUtilities.Update,
		Complete:          stepUtilities.Complete,
		Fail:              stepUtilities.Fail,
	}

	for _, middleware := range ws.editMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	return args.Next()
}

// processSave processes workflow step save events
func (ws *WorkflowStep) processSave(args types.AllMiddlewareArgs) error {
	stepUtilities := ws.createStepUtilities(args)

	middlewareArgs := WorkflowStepSaveMiddlewareArgs{
		AllMiddlewareArgs: args,
		Step:              nil, // Would be populated from actual event
		Body:              nil, // Would be populated from actual body
		View:              nil, // Would be populated from actual view
		Update:            stepUtilities.Update,
		Complete:          stepUtilities.Complete,
		Fail:              stepUtilities.Fail,
	}

	for _, middleware := range ws.saveMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	return args.Next()
}

// processExecute processes workflow step execute events
func (ws *WorkflowStep) processExecute(args types.AllMiddlewareArgs) error {
	stepUtilities := ws.createStepUtilities(args)

	middlewareArgs := WorkflowStepExecuteMiddlewareArgs{
		AllMiddlewareArgs: args,
		Step:              nil, // Would be populated from actual event
		Body:              nil, // Would be populated from actual body
		Event:             nil, // Would be populated from actual event
		Complete:          stepUtilities.Complete,
		Fail:              stepUtilities.Fail,
	}

	for _, middleware := range ws.executeMiddleware {
		if err := middleware(middlewareArgs); err != nil {
			return err
		}
	}

	return args.Next()
}

// StepUtilities contains utility functions for workflow steps
type StepUtilities struct {
	Configure StepConfigureFn
	Update    StepUpdateFn
	Complete  StepCompleteFn
	Fail      StepFailFn
}

// createStepUtilities creates utility functions for workflow step middleware
func (ws *WorkflowStep) createStepUtilities(args types.AllMiddlewareArgs) StepUtilities {
	return StepUtilities{
		Configure: func(configArgs StepConfigureArguments) error {
			// This would call the Slack API to configure the step
			return nil
		},
		Update: func(updateArgs *StepUpdateArguments) error {
			// This would call the Slack API to update the step
			return nil
		},
		Complete: func(completeArgs *StepCompleteArguments) error {
			// This would call the Slack API to complete the step
			return nil
		},
		Fail: func(failArgs StepFailArguments) error {
			// This would call the Slack API to fail the step
			return nil
		},
	}
}

// Helper functions for working with workflow step events

// IsWorkflowStepEvent checks if an event is a workflow step event
func IsWorkflowStepEvent(body []byte) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}

	if eventType, exists := parsed["type"]; exists {
		if typeStr, ok := eventType.(string); ok {
			return typeStr == "workflow_step_edit" ||
				typeStr == "view_submission" ||
				typeStr == "workflow_step_execute"
		}
	}

	// Check for workflow_step_execute event
	if event, exists := parsed["event"]; exists {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if eventType, exists := eventMap["type"]; exists {
				if typeStr, ok := eventType.(string); ok {
					return typeStr == "workflow_step_execute"
				}
			}
		}
	}

	return false
}

// ExtractCallbackID extracts callback ID from workflow step event
func ExtractCallbackID(body []byte) (string, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}

	if callbackID, exists := parsed["callback_id"]; exists {
		if callbackIDStr, ok := callbackID.(string); ok {
			return callbackIDStr, nil
		}
	}

	// Check in workflow step
	if workflowStep, exists := parsed["workflow_step"]; exists {
		if stepMap, ok := workflowStep.(map[string]interface{}); ok {
			if callbackID, exists := stepMap["callback_id"]; exists {
				if callbackIDStr, ok := callbackID.(string); ok {
					return callbackIDStr, nil
				}
			}
		}
	}

	return "", errors.NewWorkflowStepInitializationError("could not extract callback_id from workflow step event")
}
