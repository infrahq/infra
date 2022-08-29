package querylinter

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

var Analyzer = &analysis.Analyzer{
	Name: "queryBuilderLinter",
	Doc:  "checks for unsafe use of querybuilder.Builder",
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
	constructorName           = "NewQuery"
	buildFuncName             = "B"
	pkgName                   = "querybuilder"
	buildFuncReceiverTypeName = "*github.com/infrahq/infra/internal/server/data/querybuilder.Builder"
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

	if !isBuildMethod(pass, fnSe) {
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

	if xIdent, ok := se.X.(*ast.Ident); ok {
		if xIdent.Name != pkgName {
			return
		}
	}

	if callExpr, ok := cursor.Parent().(*ast.CallExpr); ok {
		if callExpr.Fun == se {
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

	if !isBuildMethod(pass, se) {
		return
	}

	if callExpr, ok := cursor.Parent().(*ast.CallExpr); ok {
		if callExpr.Fun == se {
			return
		}
	}

	pass.Reportf(se.Sel.Pos(), "Builder.%v must be called directly, not assigned to a variable or passed to a function",
		buildFuncName)
}

func isBuildMethod(pass *analysis.Pass, se *ast.SelectorExpr) bool {
	if se.Sel.Name != buildFuncName {
		return false
	}

	selection := pass.TypesInfo.Selections[se]
	if selection == nil || selection.Recv() == nil {
		// not a method call
		return false
	}

	return selection.Recv().String() == buildFuncReceiverTypeName
}
