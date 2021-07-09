package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormattedVersionRemovesV(t *testing.T) {
	Version = "v0.1-example"
	assert.Equal(t, "0.1-example", GetFormattedVersion())
}
