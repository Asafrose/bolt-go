package test

import (
	"errors"
	"testing"

	bolterrors "github.com/Asafrose/bolt-go/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	t.Run("has errors matching codes", func(t *testing.T) {
		// Create a map of error codes to their corresponding error instances
		errorMap := map[bolterrors.ErrorCode]bolterrors.CodedError{
			bolterrors.AppInitializationErrorCode:      bolterrors.NewAppInitializationError("test"),
			bolterrors.AuthorizationErrorCode:          bolterrors.NewAuthorizationError("auth failed", errors.New("auth failed")),
			bolterrors.ContextMissingPropertyErrorCode: bolterrors.NewContextMissingPropertyError("foo", "can't find foo"),
			bolterrors.ReceiverAuthenticityErrorCode:   bolterrors.NewReceiverAuthenticityError("test"),
			bolterrors.ReceiverMultipleAckErrorCode:    bolterrors.NewReceiverMultipleAckError(),
			bolterrors.UnknownErrorCode:                bolterrors.NewUnknownError(errors.New("It errored")),
		}

		for code, err := range errorMap {
			assert.Equal(t, code, err.Code(), "Error code should match for %s", code)
		}
	})

	t.Run("wraps non-coded errors", func(t *testing.T) {
		plainError := errors.New("AHH!")
		codedError := bolterrors.AsCodedError(plainError)

		// Should be wrapped in UnknownError
		unknownError, ok := codedError.(*bolterrors.UnknownError)
		assert.True(t, ok, "Should be wrapped in UnknownError")
		assert.Equal(t, bolterrors.UnknownErrorCode, unknownError.Code())
		assert.Equal(t, plainError, unknownError.Original())
	})

	t.Run("passes coded errors through", func(t *testing.T) {
		originalError := bolterrors.NewAppInitializationError("test")
		passedError := bolterrors.AsCodedError(originalError)

		// Should be the same instance
		assert.Equal(t, originalError, passedError, "Coded errors should pass through unchanged")
	})
}
