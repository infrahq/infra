package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultVersion(t *testing.T) {
	assert.Equal(t, Version, GetFormattedVersion())
}

func TestFormattedVersionRemovesV(t *testing.T) {
	Version = "v0.1-example"
	assert.Equal(t, "0.1-example", GetFormattedVersion())
}
