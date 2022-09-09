package querylinter

import (
	"os"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"gotest.tools/v3/assert"
)

func TestAnalyzer(t *testing.T) {
	files := map[string]string{
		"github.com/infrahq/infra/example/problems.go":                          readFile(t, "testdata/problems.go"),
		"github.com/infrahq/infra/internal/server/data/querybuilder/builder.go": readFile(t, "../../server/data/querybuilder/builder.go"),
	}

	dir, cleanup, err := analysistest.WriteFiles(files)
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	analysistest.Run(t, dir, Analyzer, "github.com/infrahq/infra/example")
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	raw, err := os.ReadFile(p)
	assert.NilError(t, err)
	return string(raw)
}
