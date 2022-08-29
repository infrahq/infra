package querylinter

import (
	"os"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestAnalyzer_WithDataPkg(t *testing.T) {
	c := icmd.Command("go", "run", "./cmd", "../../server/data")
	icmd.RunCmd(c).Assert(t, icmd.Success)
}

func TestAnalyzer(t *testing.T) {
	files := map[string]string{
		"github.com/infrahq/infra/example/problems.go":                          string(golden.Get(t, "problems.go")),
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
