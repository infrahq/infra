package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandStringNegativeLen(t *testing.T) {
	assert.Equal(t, RandString(-1), "")
}

func TestRandStringLen(t *testing.T) {
	assert.Equal(t, len(RandString(20)), 20)
}
