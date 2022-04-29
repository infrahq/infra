package cmd

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUserFacingError(t *testing.T) {
	var err error = Error{
		OriginalError: fmt.Errorf("underlying"),
		Message:       "failed to logout",
	}

	var e Error
	ok := errors.As(err, &e)
	assert.Assert(t, ok)
	assert.Equal(t, e.Message, "failed to logout")

	// // err != UserFacingError{} -> false
	// ok = errors.Is(err, UserFacingError{})
	// assert.Assert(t, ok)
}
