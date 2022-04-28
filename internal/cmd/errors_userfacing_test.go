package cmd

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUserFacingError(t *testing.T) {
	var err error = UserFacingError{
		Underlying:        fmt.Errorf("underlying"),
		UserFacingMessage: "failed to logout",
	}

	var userErr UserFacingError
	ok := errors.As(err, &userErr)
	assert.Assert(t, ok)
	assert.Equal(t, userErr.UserFacingMessage, "failed to logout")

	// // err != UserFacingError{} -> false
	// ok = errors.Is(err, UserFacingError{})
	// assert.Assert(t, ok)
}
