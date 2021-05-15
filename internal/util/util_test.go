package util

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestRandStringNegativeLen(t *testing.T) {
	assert.Equal(t, RandString(-1), "")
}

func TestRandStringLen(t *testing.T) {
	assert.Equal(t, len(RandString(20)), 20)
}
