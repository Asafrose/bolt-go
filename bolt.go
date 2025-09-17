// Package bolt provides a framework for building Slack apps in Go.
// This is a port of the official Slack Bolt framework from JavaScript/TypeScript.
package bolt

import (
	"github.com/Asafrose/bolt-go/pkg/app"
	"github.com/Asafrose/bolt-go/pkg/assistant"
	"github.com/Asafrose/bolt-go/pkg/conversation"
	"github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/Asafrose/bolt-go/pkg/functions"
	"github.com/Asafrose/bolt-go/pkg/helpers"
	"github.com/Asafrose/bolt-go/pkg/middleware"
	"github.com/Asafrose/bolt-go/pkg/receivers"
	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/Asafrose/bolt-go/pkg/workflow"
)

// Re-export main types and functions for convenience

// App types
type App = app.App
type AppOptions = app.AppOptions
type AuthorizeFunc = app.AuthorizeFunc
type AuthorizeSourceData = app.AuthorizeSourceData
type AuthorizeResult = app.AuthorizeResult
type ErrorHandler = app.ErrorHandler
type ExtendedErrorHandler = app.ExtendedErrorHandler
type LogLevel = types.LogLevel

// App constructor
var New = app.New

// Type definitions
type Context = types.Context
type Middleware[T any] = types.Middleware[T]
type NextFn = types.NextFn
type SayFn = types.SayFn
type RespondFn = types.RespondFn
type AckFn[T any] = types.AckFn[T]

// Middleware argument types
type AllMiddlewareArgs = types.AllMiddlewareArgs
type SlackEventMiddlewareArgs = types.SlackEventMiddlewareArgs
type SlackActionMiddlewareArgs = types.SlackActionMiddlewareArgs
type SlackCommandMiddlewareArgs = types.SlackCommandMiddlewareArgs
type SlackShortcutMiddlewareArgs = types.SlackShortcutMiddlewareArgs
type SlackViewMiddlewareArgs = types.SlackViewMiddlewareArgs
type SlackOptionsMiddlewareArgs = types.SlackOptionsMiddlewareArgs
type SlackCustomFunctionMiddlewareArgs = types.SlackCustomFunctionMiddlewareArgs

// Middleware options types
type SlackEventMiddlewareArgsOptions = middleware.SlackEventMiddlewareArgsOptions

// Constraint types
type ActionConstraints = types.ActionConstraints
type EventConstraints = types.EventConstraints
type CommandConstraints = types.CommandConstraints
type ShortcutConstraints = types.ShortcutConstraints
type ViewConstraints = types.ViewConstraints
type OptionsConstraints = types.OptionsConstraints

// Event types
type SlackAction = types.SlackAction
type BlockAction = types.BlockAction
type InteractiveMessage = types.InteractiveMessage
type DialogSubmitAction = types.DialogSubmitAction
type WorkflowStepEdit = types.WorkflowStepEdit

type SlashCommand = types.SlashCommand
type CommandResponse = types.CommandResponse

type SlackShortcut = types.SlackShortcut
type GlobalShortcut = types.GlobalShortcut
type MessageShortcut = types.MessageShortcut

type SlackView = types.SlackView
type ViewSubmission = types.ViewSubmission
type ViewClosed = types.ViewClosed
type ViewResponse = types.ViewResponse

type OptionsRequest = types.OptionsRequest
type OptionsResponse = types.OptionsResponse
type Option = types.Option
type OptionGroup = types.OptionGroup
type TextObject = types.TextObject

// Receiver types
type Receiver = types.Receiver
type ReceiverEvent = types.ReceiverEvent
type ReceiverEndpoints = types.ReceiverEndpoints
type HTTPReceiverOptions = types.HTTPReceiverOptions
type SocketModeReceiverOptions = types.SocketModeReceiverOptions
type AwsLambdaReceiverOptions = types.AwsLambdaReceiverOptions

// Receiver constructors
var NewHTTPReceiver = receivers.NewHTTPReceiver
var NewSocketModeReceiver = receivers.NewSocketModeReceiver

// Assistant types
type Assistant = assistant.Assistant
type AssistantConfig = assistant.AssistantConfig
type AssistantThreadContext = assistant.AssistantThreadContext
type AssistantThreadContextStore = assistant.AssistantThreadContextStore

type AssistantThreadStartedMiddleware = assistant.AssistantThreadStartedMiddleware
type AssistantThreadContextChangedMiddleware = assistant.AssistantThreadContextChangedMiddleware
type AssistantUserMessageMiddleware = assistant.AssistantUserMessageMiddleware

type AssistantThreadStartedMiddlewareArgs = assistant.AssistantThreadStartedMiddlewareArgs
type AssistantThreadContextChangedMiddlewareArgs = assistant.AssistantThreadContextChangedMiddlewareArgs
type AssistantUserMessageMiddlewareArgs = assistant.AssistantUserMessageMiddlewareArgs

// Assistant constructor
var NewAssistant = assistant.NewAssistant
var NewDefaultThreadContextStore = assistant.NewDefaultThreadContextStore

// Conversation types
type ConversationStore = conversation.ConversationStore
type MemoryStore = conversation.MemoryStore

// Conversation constructors (note: these are generic functions requiring type parameters)
// Use conversation.NewMemoryStore[YourType]() and conversation.ConversationContext[YourType](store)

// WorkflowStep types (deprecated)
type WorkflowStep = workflow.WorkflowStep
type WorkflowStepConfig = workflow.WorkflowStepConfig

type StepConfigureArguments = workflow.StepConfigureArguments
type StepUpdateArguments = workflow.StepUpdateArguments
type StepCompleteArguments = workflow.StepCompleteArguments
type StepFailArguments = workflow.StepFailArguments

type WorkflowStepEditMiddleware = workflow.WorkflowStepEditMiddleware
type WorkflowStepSaveMiddleware = workflow.WorkflowStepSaveMiddleware
type WorkflowStepExecuteMiddleware = workflow.WorkflowStepExecuteMiddleware

type WorkflowStepEditMiddlewareArgs = workflow.WorkflowStepEditMiddlewareArgs
type WorkflowStepSaveMiddlewareArgs = workflow.WorkflowStepSaveMiddlewareArgs
type WorkflowStepExecuteMiddlewareArgs = workflow.WorkflowStepExecuteMiddlewareArgs

// WorkflowStep constructor (deprecated)
var NewWorkflowStep = workflow.NewWorkflowStep

// Custom function types
type CustomFunction = functions.CustomFunction
type CustomFunctionOptions = functions.CustomFunctionOptions

// Custom function constructors
var NewCustomFunctionWithMiddleware = functions.NewCustomFunctionWithMiddleware

// Error types
type CodedError = errors.CodedError
type ErrorCode = errors.ErrorCode

// Error constructors
var NewAppInitializationError = errors.NewAppInitializationError
var NewAssistantInitializationError = errors.NewAssistantInitializationError
var NewAssistantMissingPropertyError = errors.NewAssistantMissingPropertyError
var NewAuthorizationError = errors.NewAuthorizationError
var NewContextMissingPropertyError = errors.NewContextMissingPropertyError
var NewInvalidCustomPropertyError = errors.NewInvalidCustomPropertyError
var NewReceiverMultipleAckError = errors.NewReceiverMultipleAckError
var NewReceiverAuthenticityError = errors.NewReceiverAuthenticityError
var NewHTTPReceiverDeferredRequestError = errors.NewHTTPReceiverDeferredRequestError
var NewMultipleListenerError = errors.NewMultipleListenerError
var NewWorkflowStepInitializationError = errors.NewWorkflowStepInitializationError

// Error utilities
var IsCodedError = errors.IsCodedError
var AsCodedError = errors.AsCodedError

// Helper types
type IncomingEventType = helpers.IncomingEventType
type EventTypeAndConversation = helpers.EventTypeAndConversation

// Helper functions
var GetTypeAndConversation = helpers.GetTypeAndConversation
var IsBodyWithTypeEnterpriseInstall = helpers.IsBodyWithTypeEnterpriseInstall
var IsEventTypeToSkipAuthorize = helpers.IsEventTypeToSkipAuthorize
var ExtractEventType = helpers.ExtractEventType
var CreateSayFunction = helpers.CreateSayFunction
var CreateRespondFunction = helpers.CreateRespondFunction
var MatchesPattern = helpers.MatchesPattern
var ExtractUserID = helpers.ExtractUserID

// Middleware functions
var OnlyActions = middleware.OnlyActions
var OnlyShortcuts = middleware.OnlyShortcuts
var OnlyCommands = middleware.OnlyCommands
var OnlyEvents = middleware.OnlyEvents
var OnlyOptions = middleware.OnlyOptions
var OnlyViewActions = middleware.OnlyViewActions
var MatchEventType = middleware.MatchEventType
var MatchCommandName = middleware.MatchCommandName
var MatchConstraints = middleware.MatchConstraints
var MatchMessage = middleware.MatchMessage
var IgnoreSelf = middleware.IgnoreSelf
var AutoAcknowledge = middleware.AutoAcknowledge
var DirectMention = middleware.DirectMention
var Subtype = middleware.Subtype
var MatchCallbackId = middleware.MatchCallbackId
var IsSlackEventMiddlewareArgsOptions = middleware.IsSlackEventMiddlewareArgsOptions

// Constants
const (
	LogLevelDebug = types.LogLevelDebug
	LogLevelInfo  = types.LogLevelInfo
	LogLevelWarn  = types.LogLevelWarn
	LogLevelError = types.LogLevelError
)

const (
	IncomingEventTypeEvent      = helpers.IncomingEventTypeEvent
	IncomingEventTypeAction     = helpers.IncomingEventTypeAction
	IncomingEventTypeCommand    = helpers.IncomingEventTypeCommand
	IncomingEventTypeOptions    = helpers.IncomingEventTypeOptions
	IncomingEventTypeViewAction = helpers.IncomingEventTypeViewAction
	IncomingEventTypeShortcut   = helpers.IncomingEventTypeShortcut
)

// Error codes
const (
	AppInitializationErrorCode             = errors.AppInitializationErrorCode
	AssistantInitializationErrorCode       = errors.AssistantInitializationErrorCode
	AssistantMissingPropertyErrorCode      = errors.AssistantMissingPropertyErrorCode
	AuthorizationErrorCode                 = errors.AuthorizationErrorCode
	ContextMissingPropertyErrorCode        = errors.ContextMissingPropertyErrorCode
	InvalidCustomPropertyErrorCode         = errors.InvalidCustomPropertyErrorCode
	CustomRouteInitializationErrorCode     = errors.CustomRouteInitializationError
	ReceiverMultipleAckErrorCode           = errors.ReceiverMultipleAckErrorCode
	ReceiverAuthenticityErrorCode          = errors.ReceiverAuthenticityErrorCode
	ReceiverInconsistentStateErrorCode     = errors.ReceiverInconsistentStateError
	MultipleListenerErrorCode              = errors.MultipleListenerErrorCode
	HTTPReceiverDeferredRequestErrorCode   = errors.HTTPReceiverDeferredRequestErrorCode
	UnknownErrorCode                       = errors.UnknownErrorCode
	WorkflowStepInitializationErrorCode    = errors.WorkflowStepInitializationErrorCode
	CustomFunctionInitializationErrorCode  = errors.CustomFunctionInitializationErrorCode
	CustomFunctionCompleteSuccessErrorCode = errors.CustomFunctionCompleteSuccessErrorCode
	CustomFunctionCompleteFailErrorCode    = errors.CustomFunctionCompleteFailErrorCode
)
