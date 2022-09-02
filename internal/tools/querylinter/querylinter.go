package querylinter

import (
	"fmt"
	"go/ast"
	"go/token"

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
			checkColumnMethodImplementations(pass, node)
			checkTableMethodImplementations(pass, node)
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
	columnsMethodName         = "Columns"
	tableMethodName           = "Table"
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

	switch arg := call.Args[0].(type) {
	case *ast.BasicLit:
		return nil
	case *ast.CallExpr:
		if id, ok := arg.Fun.(*ast.Ident); ok {
			switch id.Name {
			case "columnsForSelect", "columnsForInsert", "placeholderForColumns", "columnsForUpdate":
				// these functions should all accept a Table method, and only add the value of
				// Columns, which we check to ensure always return string literals.
				return nil
			}
		}

		if isTableMethod(arg) {
			return nil
		}
	}

	pass.Reportf(call.Pos(), "argument to Builder.%v must be a string literal, not %T",
		buildFuncName, call.Args[0])
	return nil
}

func isTableMethod(callExpr *ast.CallExpr) bool {
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != tableMethodName {
		return false
	}
	if len(callExpr.Args) != 0 {
		return false
	}
	return true
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

func checkColumnMethodImplementations(pass *analysis.Pass, node ast.Node) {
	decl, ok := node.(*ast.FuncDecl)
	if !ok {
		return
	}

	if decl.Name == nil || decl.Name.Name != columnsMethodName {
		return
	}

	if !isColumnsExpectedSignature(decl) {
		return
	}

	if count := len(decl.Body.List); count != 1 {
		pass.ReportRangef(decl.Body, "Columns method must only return a single slice literal")
		return
	}

	ret, ok := decl.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		pass.Reportf(decl.Body.List[0].Pos(), "Columns method must return as only expression")
		return
	}

	lit, ok := ret.Results[0].(*ast.CompositeLit)
	if !ok {
		pass.Reportf(ret.Results[0].Pos(), "Columns method must return a slice literal")
		return
	}

	for _, element := range lit.Elts {
		stringLit, ok := element.(*ast.BasicLit)
		if !ok || stringLit.Kind != token.STRING {
			pass.Reportf(element.Pos(), "Columns method return value must contain only string literals")
		}
	}
	return
}

func isColumnsExpectedSignature(decl *ast.FuncDecl) bool {
	if len(decl.Recv.List) != 1 {
		// not a method
		return false
	}

	if len(decl.Type.Params.List) != 0 || len(decl.Type.Results.List) != 1 {
		// wrong number of parameters or return values
		return false
	}

	arrayType, ok := decl.Type.Results.List[0].Type.(*ast.ArrayType)
	if !ok {
		return false
	}

	ident, ok := arrayType.Elt.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "string"
}

func checkTableMethodImplementations(pass *analysis.Pass, node ast.Node) {
	decl, ok := node.(*ast.FuncDecl)
	if !ok {
		return
	}

	if decl.Name == nil || decl.Name.Name != tableMethodName {
		return
	}

	if !isTableExpectedSignature(decl) {
		return
	}

	if count := len(decl.Body.List); count != 1 {
		pass.ReportRangef(decl.Body, "Table method must only return a string literal")
		return
	}

	ret, ok := decl.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		pass.Reportf(decl.Body.List[0].Pos(), "Table method must return as only expression")
		return
	}

	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		pass.Reportf(ret.Results[0].Pos(), "Table method must return a string literal")
		return
	}
	return
}

func isTableExpectedSignature(decl *ast.FuncDecl) bool {
	if len(decl.Recv.List) != 1 {
		// not a method
		return false
	}

	if len(decl.Type.Params.List) != 0 || len(decl.Type.Results.List) != 1 {
		// wrong number of parameters or return values
		return false
	}

	ident, ok := decl.Type.Results.List[0].Type.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "string"
}
