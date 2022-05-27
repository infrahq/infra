package cmd

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestCLIError(t *testing.T) {

	t.Run("only OriginalError", func(t *testing.T) {
		var err error = Error{
			OriginalError: fmt.Errorf("failed, original error"),
		}

		assert.Error(t, err, "Internal error:\nfailed, original error")
	})

	t.Run("only message", func(t *testing.T) {
		var err error = Error{
			Message: "failed, message",
		}

		assert.Error(t, err, "failed, message")
	})

	t.Run("message and error", func(t *testing.T) {
		var err error = Error{
			OriginalError: fmt.Errorf("failed, original error"),
			Message:       "failed, message",
		}

		assert.Error(t, err, "failed, message:\nfailed, original error")
	})

	t.Run("message and error, message needs formatting", func(t *testing.T) {
		var err error = Error{
			OriginalError: fmt.Errorf("failed, original error"),
			Message:       "failed, message.",
		}

		assert.Error(t, err, "failed, message:\nfailed, original error")
	})

	t.Run("message and error, message needs formatting", func(t *testing.T) {

		var err error = Error{
			OriginalError: fmt.Errorf("failed, original error"),
			Message:       "failed, message.",
		}

		assert.Error(t, err, "failed, message:\nfailed, original error")
	})

	t.Run("unwrap", func(t *testing.T) {
		var originalError = fmt.Errorf("failed, original error")

		var err error = Error{
			OriginalError: originalError,
			Message:       "failed, message",
		}

		ok := errors.Is(err, originalError)
		assert.Assert(t, ok)
	})
}
