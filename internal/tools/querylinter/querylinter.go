package querylinter

import (
	"flag"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

var Analyzer = &analysis.Analyzer{
	Name:  "queryBuilderLinter",
	Doc:   "checks for unsafe use of querybuilder.Builder",
	Flags: flag.FlagSet{},
	Run: func(pass *analysis.Pass) (interface{}, error) {
		return nil, run(pass)
	},
}

func run(pass *analysis.Pass) error {
	var err error

	for _, file := range pass.Files {
		inspect := func(cursor *astutil.Cursor) bool {
			node := cursor.Node()
			if node == nil {
				return true
			}
			if err = checkNewQuery(pass, node); err != nil {
				return false
			}
			if err = checkB(pass, node); err != nil {
				return false
			}

			checkConstructorNotACallExpr(pass, cursor)
			checkBNotACallExpr(pass, cursor)
			return true
		}
		astutil.Apply(file, inspect, nil)
	}
	return err
}

var (
	constructorName = "NewQuery"
	buildFuncName   = "B"
)

func checkNewQuery(pass *analysis.Pass, node ast.Node) error {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil
	}

	se, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	if se.Sel.Name != constructorName {
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
		pass.Reportf(call.Pos(), "argument to Builder.%v must be a string literal, not %T",
			buildFuncName, call.Args[0])
		return nil
	}
	return nil
}

func checkConstructorNotACallExpr(pass *analysis.Pass, cursor *astutil.Cursor) {
	se, ok := cursor.Node().(*ast.SelectorExpr)
	if !ok {
		return
	}

	if se.Sel.Name != constructorName {
		return
	}

	switch parent := cursor.Parent().(type) {
	case *ast.CallExpr:
		if parent.Fun == se {
			return
		}
	}

	pass.Reportf(se.Sel.Pos(), "%v must be called directly, not assigned to a variable or passed to a function",
		constructorName)
}

func checkBNotACallExpr(pass *analysis.Pass, cursor *astutil.Cursor) {
	se, ok := cursor.Node().(*ast.SelectorExpr)
	if !ok {
		return
	}

	if se.Sel.Name != buildFuncName {
		return
	}

	switch parent := cursor.Parent().(type) {
	case *ast.CallExpr:
		if parent.Fun == se {
			return
		}
	}

	pass.Reportf(se.Sel.Pos(), "Builder.%v must be called directly, not assigned to a variable or passed to a function",
		buildFuncName)
}
