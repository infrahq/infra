package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/infrahq/infra/internal/tools/querylinter"
)

func main() {
	singlechecker.Main(querylinter.Analyzer)
}

type analyzerPlugin struct{}

func (analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{querylinter.Analyzer}
}

// AnalyzerPlugin implements the interface for golangci-lint plugins
// nolint
var AnalyzerPlugin = analyzerPlugin{}
