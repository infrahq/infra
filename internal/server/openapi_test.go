package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteOpenAPISpec is not really a test. It's a way of ensuring the openapi
// spec is updated.
// TODO: replace this with a test that uses golden, and a CI check to make sure the
// file in git matches the source code.
func TestWriteOpenAPISpec(t *testing.T) {
	s := Server{}
	s.GenerateRoutes()

	filename := "../../docs/api/openapi3.json"
	err := writeOpenAPISpecToFile(filename)
	require.NoError(t, err)
}
