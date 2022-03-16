package server

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/infrahq/infra/api"
)

var openAPISchema = openapi3.T{}
var pathIDReplacer = regexp.MustCompile(`:\w+`)

func register[Req, Res any](method, basePath, path string, handler ReqResHandlerFunc[Req, Res]) {
	funcName := getFuncName(handler)

	rqt := reflect.TypeOf(*new(Req))
	rst := reflect.TypeOf(*new(Res))

	reg(method, basePath, path, funcName, rqt, rst)
}

func registerReq[Req any](method, basePath, path string, handler ReqHandlerFunc[Req]) {
	funcName := getFuncName(handler)

	rqt := reflect.TypeOf(*new(Req))
	rst := reflect.TypeOf(nil)

	reg(method, basePath, path, funcName, rqt, rst)
}

func registerRes[Res any](method, basePath, path string, handler ResHandlerFunc[Res]) {
	funcName := getFuncName(handler)

	rqt := reflect.TypeOf(nil)
	rst := reflect.TypeOf(*new(Res))

	reg(method, basePath, path, funcName, rqt, rst)
}

func reg(method, basePath, path, funcName string, rqt, rst reflect.Type) {
	path = strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(path, "/")
	path = pathIDReplacer.ReplaceAllStringFunc(path, func(s string) string {
		return "{" + strings.TrimLeft(s, ":") + "}"
	})

	if openAPISchema.Components.Schemas == nil {
		openAPISchema.Components.Schemas = openapi3.Schemas{}
	}
	if openAPISchema.Paths == nil {
		openAPISchema.Paths = openapi3.Paths{}
	}

	p, ok := openAPISchema.Paths[path]
	if !ok {
		p = &openapi3.PathItem{}
	}

	op := openapi3.NewOperation()
	op.OperationID = funcName
	op.Description = funcName
	op.Summary = funcName
	if rqt != nil {
		buildRequest(rqt, op)
	}
	op.Responses = buildResponse(rst)

	restMethods := []string{"Get", "List", "Delete", "Create", "Update"}
tagLoop:
	for _, method := range restMethods {
		if strings.HasPrefix(funcName, method) {
			tagName := strings.TrimPrefix(funcName, method)
			// depluralize
			tagName = strings.TrimSuffix(tagName, "s")
			for _, tag := range op.Tags {
				if tag == tagName {
					break tagLoop
				}
			}
			op.Tags = append(op.Tags, tagName)
		}
	}

	switch funcName {
	case "Login", "Logout":
		op.Tags = append(op.Tags, "Authentication")
	}

	if len(op.Tags) == 0 {
		op.Tags = append(op.Tags, "Misc")
	}

	switch method {
	case "GET":
		p.Get = op
	case "PATCH":
		p.Patch = op
	case "POST":
		p.Post = op
	case "PUT":
		p.Put = op
	case "DELETE":
		p.Delete = op
	default:
		panic("unexpected http method " + method)
	}

	openAPISchema.Paths[path] = p
}

func getFuncName(i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	nameParts := strings.Split(name, ".")
	name = nameParts[len(nameParts)-1]
	name = strings.TrimSuffix(name, "-fm")
	return name
}

func createComponent(rst reflect.Type) *openapi3.SchemaRef {
	if openAPISchema.Components.Schemas == nil {
		openAPISchema.Components.Schemas = openapi3.Schemas{}
	}

	switch rst.Kind() {
	case reflect.Pointer:
		return createComponent(rst.Elem())
	case reflect.Slice:
		schema := createComponent(rst.Elem())

		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:  "array",
				Items: schema,
			},
		}
	case reflect.Struct:
		schema := &openapi3.Schema{
			Properties: openapi3.Schemas{},
		}

		for i := 0; i < rst.NumField(); i++ {
			f := rst.Field(i)
			schema.Properties[getFieldName(f, rst)] = buildProperty(f, f.Type, rst, schema)
		}

		if _, ok := openAPISchema.Components.Schemas[rst.Name()]; ok {
			return &openapi3.SchemaRef{
				Ref: "#/components/schemas/" + rst.Name(),
			}
		}

		schemaRef := &openapi3.SchemaRef{
			Value: schema,
		}

		openAPISchema.Components.Schemas[rst.Name()] = schemaRef

		return &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + rst.Name(),
		}
	default:
		panic("unexpected component kind " + rst.Kind().String())
	}
}

func buildProperty(f reflect.StructField, t, parent reflect.Type, parentSchema *openapi3.Schema) *openapi3.SchemaRef {
	if t.Kind() == reflect.Pointer {
		return buildProperty(f, t.Elem(), parent, parentSchema)
	}

	s := &openapi3.Schema{}
	setTypeInfo(t, s)
	setTagInfo(f, t, parent, s, parentSchema)

	if s.Type == "array" {
		s.Items = buildProperty(f, t.Elem(), parent, parentSchema)
	}

	if s.Type == "object" {
		s.Properties = openapi3.Schemas{}

		for i := 0; i < t.NumField(); i++ {
			f2 := t.Field(i)
			s.Properties[f2.Name] = buildProperty(f2, f2.Type, t, s)
		}
	}

	return &openapi3.SchemaRef{
		Value: s,
	}
}

func isDev() bool {
	dirs := []string{".git", "docs"}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			return false
		}
		if !info.IsDir() {
			return false
		}
	}

	return true
}

func generateOpenAPI() {
	if !isDev() {
		return
	}

	openAPISchema.OpenAPI = "3.0.0"
	openAPISchema.Info = &openapi3.Info{
		Title:       "Infra API",
		Version:     time.Now().Format("2006-01-02"),
		Description: "Infra API",
		License:     &openapi3.License{Name: "Apache 2.0", URL: "https://www.apache.org/licenses/LICENSE-2.0.html"},
	}
	openAPISchema.Servers = append(openAPISchema.Servers, &openapi3.Server{
		URL: "https://api.infrahq.com",
	})

	// json
	f2, err := os.Create("docs/api/openapi3.json")
	if err != nil {
		panic(err)
	}
	defer f2.Close()
	enc2 := json.NewEncoder(f2)
	enc2.SetIndent("", "  ")
	if err := enc2.Encode(openAPISchema); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func setTagInfo(f reflect.StructField, t, parent reflect.Type, schema, parentSchema *openapi3.Schema) {
	if ex := getDefaultExampleForType(t); len(ex) > 0 {
		schema.Example = ex
	}

	if example, ok := f.Tag.Lookup("example"); ok {
		schema.Example = example
	}
	if validate, ok := f.Tag.Lookup("validate"); ok {
		for _, val := range strings.Split(validate, ",") {
			if val == "required" && parentSchema != nil {
				parentSchema.Required = append(parentSchema.Required, getFieldName(f, parent))
			}
			if val == "email" {
				schema.Example = "email@example.com"
			}
		}
	}

}

var exampleTime = time.Date(2022, 3, 14, 9, 48, 0, 0, time.UTC).Format(time.RFC3339)

// `type` can be one of the following only: "object", "array", "string", "number", "integer", "boolean", "null".
// `format` has a few defined types, but can be anything. https://swagger.io/docs/specification/data-models/data-types/
func setTypeInfo(t reflect.Type, schema *openapi3.Schema) {
	switch structNameWithPkg(t) {
	case "time.Time":
		schema.Type = "string"
		schema.Format = "date-time" // date-time is rfc3339
		schema.Example = exampleTime
		return
	case "uid.ID":
		schema.Type = "string"
		schema.Format = "uid"
		schema.Pattern = `[\da-zA-HJ-NP-Z]{1,11}`
		schema.Example = "4yJ3n3D8E2"
		return
	case "uid.PolymorphicID":
		schema.Type = "string"
		schema.Format = "poly-uid"
		schema.Pattern = `\w:[\da-zA-HJ-NP-Z]{1,11}`
		schema.Example = "u:4yJ3n3D8E3"
		return
	}

	switch t.Kind() {
	case reflect.Pointer:
		setTypeInfo(t.Elem(), schema)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		schema.Format = t.Kind().String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
		schema.Format = t.Kind().String()
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
		schema.Format = t.Kind().String()
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.String:
		schema.Type = "string"
	case reflect.Slice:
		schema.Type = "array"
	case reflect.Struct:
		schema.Type = "object"
	default:
		panic("unexpected type " + t.Kind().String())
	}
}

func pstr(s string) *string {
	return &s
}

func buildResponse(rst reflect.Type) openapi3.Responses {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{Type: "object"},
	}

	if rst != nil {
		schema = createComponent(rst)
	}

	resp := openapi3.NewResponses()
	resp["default"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: pstr("Success"),
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: schema,
				},
			},
		},
	}

	errStruct := &api.Error{}
	t := reflect.TypeOf(errStruct)
	errComp := createComponent(t)

	content := openapi3.Content{"application/json": &openapi3.MediaType{
		Schema: errComp,
	}}

	resp["400"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: pstr("Bad Request"),
			Content:     content,
		},
	}

	resp["401"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: pstr("Unauthorized: Requestor is not authenticated"),
			Content:     content,
		},
	}

	resp["403"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: pstr("Forbidden: Requestor does not have the right permissions"),
			Content:     content,
		},
	}

	resp["409"] = &openapi3.ResponseRef{ // also used for Conflict
		Value: &openapi3.Response{
			Description: pstr("Duplicate Record"),
			Content:     content,
		},
	}

	resp["404"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: pstr("Not Found"),
			Content:     content,
		},
	}

	return resp
}

func buildRequest(r reflect.Type, op *openapi3.Operation) {
	op.Parameters = openapi3.NewParameters()

	schema := &openapi3.Schema{
		Type:       "object",
		Properties: openapi3.Schemas{},
	}

	switch r.Kind() {
	case reflect.Pointer:
		buildRequest(r.Elem(), op)
		return
	case reflect.Struct:
		for i := 0; i < r.NumField(); i++ {
			f := r.Field(i)

			// check first if it's a json field
			if name, ok := f.Tag.Lookup("json"); ok {
				jsonName := strings.Split(name, ",")[0]
				if jsonName != "-" {
					prop := buildProperty(f, f.Type, r, schema)

					schema.Properties[name] = prop
					continue
				}
			}

			// if not, it's a query or uri parameter
			p := &openapi3.Parameter{
				Name:     getFieldName(f, r),
				Schema:   buildProperty(f, f.Type, r, nil),
				Required: false,
				In:       "",
			}

			if name, ok := f.Tag.Lookup("form"); ok {
				p.Name = name
				p.In = "query"
			}

			if name, ok := f.Tag.Lookup("uri"); ok {
				uriName := strings.Split(name, ",")[0]
				p.Name = uriName
				p.In = "path"
				p.Required = true
			}

			if p.In == "" {
				// field isn't properly labelled
				panic(fmt.Sprintf("field %q of struct %q must have a tag (json, form, or uri) with a name or '-'", f.Name, r.Name()))
			}

			if ex := getDefaultExampleForType(f.Type); len(ex) > 0 {
				p.Example = ex
			}

			if example, ok := f.Tag.Lookup("example"); ok {
				p.Example = example
			}

			if validate, ok := f.Tag.Lookup("validate"); ok {
				for _, val := range strings.Split(validate, ",") {
					if val == "required" {
						p.Required = true
					}
					if val == "email" {
						p.Example = "email@example.com"
					}
				}
			}

			op.AddParameter(p)
		}
	default:
		panic("unexpected type " + r.Kind().String() + "(" + r.Name() + ")")
	}

	if len(schema.Properties) > 0 {
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{
							Value: schema,
						},
					},
				},
			},
		}
	}

}

func getDefaultExampleForType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return getDefaultExampleForType(t.Elem())
	}

	name := structNameWithPkg(t)
	switch name {
	case "uid.ID":
		return "4yJ3n3D8E2"
	case "uid.PolymorphicID":
		return "u:4yJ3n3D8E3"
	case "time.Time":
		return exampleTime
	default:
		return ""
	}
}

func structNameWithPkg(t reflect.Type) string {
	path := strings.Split(t.PkgPath(), "/")
	p := path[len(path)-1]

	if len(p) > 0 {
		return p + "." + t.Name()
	}

	return t.Name()
}

func getFieldName(f reflect.StructField, parent reflect.Type) string {
	if name, ok := f.Tag.Lookup("json"); ok {
		if name != "-" {
			return strings.Split(name, ",")[0]
		}
	}

	if name, ok := f.Tag.Lookup("form"); ok {
		return name
	}

	if name, ok := f.Tag.Lookup("uri"); ok {
		return name
	}

	panic(fmt.Sprintf("field %q of struct %q must have a tag (json, form, or uri) with a name or '-'", f.Name, parent.Name()))
}
