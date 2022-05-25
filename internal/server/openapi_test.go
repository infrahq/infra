package server

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
)

// TestWriteOpenAPISpec is not really a test. It's a way of ensuring the openapi
// spec is updated when routes change.
func TestWriteOpenAPISpec(t *testing.T) {
	s := Server{}
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	filename := "../../docs/api/openapi3.json"
	err := WriteOpenAPIDocToFile(routes.OpenAPIDocument, filename)
	assert.NilError(t, err)
}
