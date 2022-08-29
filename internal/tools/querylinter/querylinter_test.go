package querylinter

import (
	"fmt"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestAnalyzer_WithDataPkg(t *testing.T) {
	c := icmd.Command("go", "run", "./cmd", "../../server/data")
	result := icmd.RunCmd(c)
	fmt.Println(result.String())
	t.Fail()
}

func TestAnalyzer(t *testing.T) {
	files := map[string]string{
		"example/problems.go": string(golden.Get(t, "problems.go")),
	}

	dir, cleanup, err := analysistest.WriteFiles(files)
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	analysistest.Run(t, dir, Analyzer, "example")
}
