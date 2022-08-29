package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/infrahq/infra/internal/tools/querylinter"
)

func main() {
	singlechecker.Main(querylinter.Analyzer)
}
