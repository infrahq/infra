package data

import (
	"flag"
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	"golang.org/x/tools/go/packages"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data/model"
)

var tables = []Table{}

type Table interface {
	Table() string
}

var flagGenerate = flag.String("generate", "",
	"generate methods for this struct, which must be in the tables list. Use 'all' to generate everything.")

// TestGenerateTableMethods is not really a test, use it to update the methods of a
// tables type.
//
//     go test -run TestGenerateTableMethods ./internal/server/data -generate=<structName>
//
// Use `-generate=all` to update all types.
//
// This automation runs as a test because reflection makes it much easier to read
// some data used for generation, and running as a test makes it easy to use reflection.
func TestGenerateTableMethods(t *testing.T) {
	inputName := *flagGenerate
	if inputName == "" {
		return
	}

	targets := targetsForGenerate(inputName)
	if len(targets) == 0 {
		t.Fatalf("no target found for generating %v", inputName)
	}

	var fileset token.FileSet
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
		Fset: &fileset,
	}
	pkgs, err := packages.Load(cfg, ".")
	assert.NilError(t, err)
	assert.Equal(t, len(pkgs), 1)
	pkg := pkgs[0]
	assert.Equal(t, pkg.Name, "data")

	for _, target := range targets {
		typ := reflect.TypeOf(target)
		pos := positionOfType(pkg, typ.Name())
		if pos == 0 {
			t.Fatalf("could not find type %v in package data", inputName)
		}

		// TODO: lookup stmt in parsed schema
		var stmt string
		desc, err := model.ParseCreateTable(stmt)
		assert.NilError(t, err)

		filename := fileset.File(pos).Name()
		err = model.GenerateMethods(target, desc.ColumnNames(), filename)
		assert.NilError(t, err)
	}
}

func targetsForGenerate(name string) []any {
	var result []any
	for _, m := range tables {
		typ := reflect.TypeOf(m)
		switch {
		case name == "all":
			result = append(result, m)
		case typ.Name() == name:
			result = append(result, m)
			return result
		}
	}
	return result
}

func positionOfType(pkg *packages.Package, name string) token.Pos {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}

			typ := gen.Specs[0].(*ast.TypeSpec) // nolint:forcetypeassert
			if typ.Name.Name == name {
				return typ.Pos()
			}
		}
	}
	return 0
}
