package server

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

// TestWriteOpenAPIDocToFile runs the OpenAPI document generation to preview the changes.
// This test is used to catch any potential problems with openapi doc generation in the PR
// that introduces them. Without this test we wouldn't notice until release time.
// To update the expected value, run:
//
//	go test ./internal/server -update
func TestWriteOpenAPIDocToFile(t *testing.T) {
	patchProductVersion(t, "0.0.0")
	s := Server{metricsRegistry: prometheus.NewRegistry()}
	routes := s.GenerateRoutes()

	filename := filepath.Join(t.TempDir(), "openapi3.json")
	err := WriteOpenAPIDocToFile(routes.OpenAPIDocument, "0.0.0", filename)
	assert.NilError(t, err)

	actual, err := ioutil.ReadFile(filename)
	assert.NilError(t, err)
	golden.Assert(t, string(actual), "../../../docs/api/openapi3.json")
}

func patchProductVersion(t *testing.T, version string) {
	orig := productVersion
	productVersion = func() string {
		return version
	}
	t.Cleanup(func() {
		productVersion = orig
	})

}
