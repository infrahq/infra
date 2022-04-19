package server

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
)

// TestWriteOpenAPISpec is not really a test. It's a way of ensuring the openapi
// spec is updated.
// TODO: replace this with a test that uses golden, and a CI check to make sure the
// file in git matches the source code.
func TestWriteOpenAPISpec(t *testing.T) {
	s := Server{}
	s.GenerateRoutes(prometheus.NewRegistry())

	filename := "../../docs/api/openapi3.json"
	err := WriteOpenAPISpecToFile(filename)
	assert.NilError(t, err)
}
