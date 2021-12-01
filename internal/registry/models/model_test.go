package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	id := NewID()

	require.Equal(t, id.Version().String(), "VERSION_1")
	require.Equal(t, id.Variant().String(), "RFC4122")
}
