package model

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

// GenerateMethods for a model struct. These methods allow the model to
// perform queries against a database table without repeating the names of
// columns and fields.
func GenerateMethods(model any, columns []string, filename string) error {
	rt := reflect.TypeOf(model)

	var fileset token.FileSet
	var mode = parser.ParseComments | parser.AllErrors
	file, err := parser.ParseFile(&fileset, filename, nil, mode)
	if err != nil {
		return fmt.Errorf("failed to source file: %w", err)
	}

	recvName := strings.ToLower(rt.Name()[:1])
	methodIndex := make(map[string]int)
	for i, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}
		if astIdentForNode(fn.Recv.List[0].Type).Name != rt.Name() {
			continue
		}

		if names := fn.Recv.List[0].Names; len(names) > 0 && names[0].Name != "_" {
			recvName = names[0].Name
		}
		methodIndex[fn.Name.Name] = i
	}

	tmpl, err := loadTemplate(&fileset)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	data := templateData{
		columns:      columns,
		receiverName: recvName,
		reflectType:  rt,
	}
	rendered, err := renderTemplate(tmpl, data)
	if err != nil {
		return err
	}

	// TODO: more sophisticated replacement of fields and columns to allow for
	// existing modification and formatting.

	var lastIndex int
	for _, idx := range methodIndex {
		if idx > lastIndex {
			lastIndex = idx
		}
	}
	lastIndex++ // insert after last

	for _, replacement := range rendered {
		if origIndex, ok := methodIndex[replacement.Name.Name]; ok {
			// nolint:forcetypeassert // already checked FuncDecl above
			file.Decls[origIndex].(*ast.FuncDecl).Body = replacement.Body
			continue
		}

		file.Decls = append(file.Decls[:lastIndex], append([]ast.Decl{replacement}, file.Decls[lastIndex:]...)...)
		lastIndex++
	}

	return writeFile(&fileset, filename, file)
}

type templateData struct {
	columns      []string
	receiverName string
	reflectType  reflect.Type
}

func loadTemplate(fileset *token.FileSet) (*ast.File, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to lookup filename of template")
	}
	tmplFile := filepath.Join(filepath.Dir(file), "model.go")
	return parser.ParseFile(fileset, tmplFile, nil, parser.ParseComments)
}

// nolint:forcetypeassert // panics are ok if the template does not match this function
func renderTemplate(tmpl *ast.File, data templateData) ([]*ast.FuncDecl, error) {
	fields, err := reflectFields(data.reflectType)
	if err != nil {
		return nil, fmt.Errorf("failed to read model struct: %w", err)
	}

	// sort by lowercase name, to hopefully match struct field names
	sort.Slice(data.columns, func(i, j int) bool {
		return strings.ToLower(data.columns[i]) < strings.ToLower(data.columns[j])
	})

	rendered := make([]*ast.FuncDecl, 3)
	for _, decl := range tmpl.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}

		// Add an empty comment group so that printed ast formats better
		fn.Doc = &ast.CommentGroup{List: []*ast.Comment{}}

		fn.Recv.List[0].Names[0].Name = data.receiverName
		astIdentForNode(fn.Recv.List[0].Type).Name = data.reflectType.Name()
		switch fn.Name.Name {
		case "Columns":
			slice := fn.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CompositeLit)
			for _, c := range data.columns {

				slice.Elts = append(slice.Elts, &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"` + c + `"`,
				})
			}
			rendered[0] = fn

		case "Values":
			slice := fn.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CompositeLit)
			for _, f := range fields {
				slice.Elts = append(slice.Elts, &ast.SelectorExpr{
					X:   &ast.Ident{Name: data.receiverName},
					Sel: &ast.Ident{Name: f.Name},
				})
			}
			rendered[1] = fn

		case "ScanFields":
			slice := fn.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CompositeLit)
			for _, f := range fields {
				slice.Elts = append(slice.Elts, &ast.UnaryExpr{
					Op: token.AND,
					X: &ast.SelectorExpr{
						X:   &ast.Ident{Name: data.receiverName},
						Sel: &ast.Ident{Name: f.Name},
					},
				})
			}
			rendered[2] = fn
		}
	}
	return rendered, nil
}

func astIdentForNode(expr ast.Expr) *ast.Ident {
	switch node := expr.(type) {
	case *ast.Ident:
		return node
	case *ast.StarExpr:
		return node.X.(*ast.Ident) // nolint:forcetypeassert
	default:
		panic(fmt.Sprintf("not sure how to get a name from a %T", expr))
	}
}

func writeFile(fileset *token.FileSet, filename string, source *ast.File) error {
	var buf bytes.Buffer
	if err := format.Node(&buf, fileset, source); err != nil {
		return fmt.Errorf("failed to format file after update: %w", err)
	}

	fh, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %v: %w", filename, err)
	}
	if _, err = fh.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write file %v: %w", filename, err)
	}
	if err := fh.Sync(); err != nil {
		return fmt.Errorf("failed to sync file %v: %w", filename, err)
	}
	return nil
}

// reflectFields returns all the exported struct fields on table. Any fields on
// embedded structs are included in the list. Fields with a struct tag of
// `db:"-"` will be excluded from the list.
func reflectFields(table any) ([]reflect.StructField, error) {
	typ, ok := table.(reflect.Type)
	if !ok {
		typ = reflect.TypeOf(table)
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("not a struct: %v", typ)
	}

	var fields []reflect.StructField
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		if tag := field.Tag.Get("db"); strings.HasPrefix(tag, "-") {
			continue
		}
		if field.Anonymous {
			embedded, err := reflectFields(field.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embedded...)
			continue
		}

		fields = append(fields, field)
	}

	// sort by lowercase name, to hopefully match table columns
	sort.Slice(fields, func(i, j int) bool {
		return strings.ToLower(fields[i].Name) < strings.ToLower(fields[j].Name)
	})
	return fields, nil
}
