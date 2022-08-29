package querylinter

import (
	"flag"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:  "queryBuilderLinter",
	Doc:   "checks for unsafe use of data.queryBuilder",
	Flags: flag.FlagSet{},
	Run: func(pass *analysis.Pass) (interface{}, error) {
		return nil, run(pass)
	},
}

func run(pass *analysis.Pass) error {
	var err error

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			if err = checkNewQuery(pass, node); err != nil {
				return false
			}
			if err = checkB(pass, node); err != nil {
				return false
			}
			return true
		})
	}
	return err
}

// TODO:
// look for newQuery or queryBuilder.B being assigned to variables
// look for direct access to queryBuilder.query

var (
	constructorName = "newQuery"
	buildFuncName   = "B"
)

func checkNewQuery(pass *analysis.Pass, node ast.Node) error {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil
	}

	fnIdent, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil
	}

	if fnIdent.Name != constructorName {
		return nil
	}

	if count := len(call.Args); count != 1 {
		return fmt.Errorf("unexpected argument count %v to %v", count, constructorName)
	}

	if _, ok := call.Args[0].(*ast.BasicLit); !ok {
		pass.Reportf(call.Pos(), "argument to %v must be a string literal, not %T",
			constructorName, call.Args[0])
		return nil
	}

	return nil
}

func checkB(pass *analysis.Pass, node ast.Node) error {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil
	}

	fnSe, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	if fnSe.Sel.Name != buildFuncName {
		return nil
	}

	if count := len(call.Args); count < 1 {
		return fmt.Errorf("unexpected argument count %v to queryBuilder.%v", count, buildFuncName)
	}

	if _, ok := call.Args[0].(*ast.BasicLit); !ok {
		pass.Reportf(call.Pos(), "argument to queryBuilder.%v must be a string literal, not %T",
			buildFuncName, call.Args[0])
		return nil
	}
	return nil
}
