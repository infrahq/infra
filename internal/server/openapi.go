package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
)

var (
	pathIDReplacer            = regexp.MustCompile(`:\w+`)
	funcPartialNameToTagNames = map[string]string{
		"Grant":       "Grants",
		"User":        "Users",
		"Group":       "Groups",
		"AccessKey":   "Authentication",
		"Provider":    "Providers",
		"Destination": "Destinations",
		"Token":       "Destinations",
		"Login":       "Authentication",
		"Logout":      "Authentication",
	}
)

// register an API endpoint that has both a request and response value.
func register[Req, Res any](a *API, method, path string, handler ReqResHandlerFunc[Req, Res]) {
	funcName := getFuncName(handler)

	//nolint:gocritic
	rqt := reflect.TypeOf(*new(Req))
	//nolint:gocritic
	rst := reflect.TypeOf(*new(Res))

	a.register(method, path, funcName, rqt, rst)
}

// registerDelete registers an API endpoint that has no response value, which is
// currently only endpoints that use the DELETE method.
func registerDelete[Req any](a *API, method, path string, handler ReqHandlerFunc[Req]) {
	funcName := getFuncName(handler)

	//nolint:gocritic
	rqt := reflect.TypeOf(*new(Req))
	rst := reflect.TypeOf(nil)

	a.register(method, path, funcName, rqt, rst)
}

// register adds the route to the API.OpenAPIDocument.
func (a *API) register(method, path, funcName string, rqt, rst reflect.Type) {
	path = pathIDReplacer.ReplaceAllStringFunc(path, func(s string) string {
		return "{" + strings.TrimLeft(s, ":") + "}"
	})

	if a.openAPIDoc.Components.Schemas == nil {
		a.openAPIDoc.Components.Schemas = openapi3.Schemas{}
	}

	if a.openAPIDoc.Paths == nil {
		a.openAPIDoc.Paths = openapi3.Paths{}
	}

	p, ok := a.openAPIDoc.Paths[path]
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

	op.Responses = buildResponse(a.openAPIDoc, rst)

tagLoop:
	for _, partialName := range orderedTagNames() {
		tagName := funcPartialNameToTagNames[partialName]
		if strings.Contains(funcName, partialName) {
			for _, tag := range op.Tags {
				if tag == tagName {
					continue tagLoop
				}
			}
			op.Tags = append(op.Tags, tagName)

		}
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

	a.openAPIDoc.Paths[path] = p
}

func getFuncName(i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	nameParts := strings.Split(name, ".")
	name = nameParts[len(nameParts)-1]
	name = strings.TrimSuffix(name, "-fm")
	return name
}

func orderedTagNames() []string {
	tagNames := make([]string, 0, len(funcPartialNameToTagNames))
	for k := range funcPartialNameToTagNames {
		tagNames = append(tagNames, k)
	}

	sort.Strings(tagNames)
	return tagNames
}

func createComponent(openAPIDoc openapi3.T, rst reflect.Type) *openapi3.SchemaRef {
	if openAPIDoc.Components.Schemas == nil {
		openAPIDoc.Components.Schemas = openapi3.Schemas{}
	}

	//nolint:exhaustive
	switch rst.Kind() {
	case reflect.Pointer:
		return createComponent(openAPIDoc, rst.Elem())
	case reflect.Slice:
		schema := createComponent(openAPIDoc, rst.Elem())

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

		name := strings.ReplaceAll(rst.Name(), rst.PkgPath()+".", "")
		name = strings.ReplaceAll(name, "[", "_")
		name = strings.ReplaceAll(name, "]", "")

		for i := 0; i < rst.NumField(); i++ {
			f := rst.Field(i)
			schema.Properties[getFieldName(f, rst)] = buildProperty(f, f.Type, rst, schema)
		}

		if _, ok := openAPIDoc.Components.Schemas[name]; ok {
			return &openapi3.SchemaRef{
				Ref: "#/components/schemas/" + name,
			}
		}

		schemaRef := &openapi3.SchemaRef{
			Value: schema,
		}

		openAPIDoc.Components.Schemas[name] = schemaRef

		return &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + name,
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
	setTagInfo(f, t, parent, s, parentSchema)
	setTypeInfo(t, s)

	if s.Type == "array" {
		s.Items = buildProperty(f, t.Elem(), parent, parentSchema)
	}

	if s.Type == "object" {
		s.Properties = openapi3.Schemas{}

		for i := 0; i < t.NumField(); i++ {
			f2 := t.Field(i)
			s.Properties[getFieldName(f2, t)] = buildProperty(f2, f2.Type, t, s)
		}
	}

	return &openapi3.SchemaRef{
		Value: s,
	}
}

func writeOpenAPISpec(spec openapi3.T, out io.Writer) error {
	spec.OpenAPI = "3.0.0"
	spec.Info = &openapi3.Info{
		Title:       "Infra API",
		Version:     internal.FullVersion(),
		Description: "Infra API",
		License:     &openapi3.License{Name: "Elastic License v2.0", URL: "https://www.elastic.co/licensing/elastic-license"},
	}
	spec.Servers = []*openapi3.Server{
		{URL: "https://api.infrahq.com"},
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}
	return nil
}

func WriteOpenAPIDocToFile(openAPIDoc openapi3.T, filename string) error {
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := writeOpenAPISpec(openAPIDoc, fh); err != nil {
		return err
	}
	return nil
}

func setTagInfo(f reflect.StructField, t, parent reflect.Type, schema, parentSchema *openapi3.Schema) {
	if ex := getDefaultExampleForType(t); len(ex) > 0 {
		schema.Example = ex
	}

	if example, ok := f.Tag.Lookup("example"); ok {
		schema.Example = example
	}

	if note, ok := f.Tag.Lookup("note"); ok {
		schema.Description = note
	}

	if validate, ok := f.Tag.Lookup("validate"); ok {
		for _, val := range strings.Split(validate, ",") {
			if val == "required" && parentSchema != nil {
				parentSchema.Required = append(parentSchema.Required, getFieldName(f, parent))
			}

			if strings.HasPrefix(val, "min=") {
				schema.MinLength = parseMinLength(val)
			}

			if strings.HasPrefix(val, "oneof=") {
				schema.Enum = parseOneOf(val)
			}
		}
	}
}

var exampleTime = time.Date(2022, 3, 14, 9, 48, 0, 0, time.UTC).Format(time.RFC3339)

// `type` can be one of the following only: "object", "array", "string", "number", "integer", "boolean", "null".
// `format` has a few defined types, but can be anything. https://swagger.io/docs/specification/data-models/data-types/
func setTypeInfo(t reflect.Type, schema *openapi3.Schema) {
	switch structNameWithPkg(t) {
	case "api.Time", "time.Time":
		schema.Type = "string"
		schema.Format = "date-time" // date-time is rfc3339
		schema.Example = exampleTime
		if len(schema.Description) == 0 {
			schema.Description = "formatted as an RFC3339 date-time"
		}
		return

	case "api.Duration", "time.Duration":
		schema.Type = "string"
		schema.Format = "duration"
		schema.Example = "72h3m6.5s"
		if len(schema.Description) == 0 {
			schema.Description = "a duration of time supporting (h)ours, (m)inutes, and (s)econds"
		}
		return

	case "uid.ID":
		schema.Type = "string"
		schema.Format = "uid"
		schema.Pattern = `[\da-zA-HJ-NP-Z]{1,11}`
		schema.Example = "4yJ3n3D8E2"
		return

	case "api.IDOrSelf":
		schema.Type = "string"
		schema.Format = "uid|self"
		schema.Pattern = `[\da-zA-HJ-NP-Z]{1,11}|self`
		schema.Example = "4yJ3n3D8E2"
		schema.Description = "a uid or the literal self"
		return

	case "uid.PolymorphicID":
		schema.Type = "string"
		schema.Format = "poly-uid"
		schema.Pattern = `\w:[\da-zA-HJ-NP-Z]{1,11}`
		schema.Example = "i:4yJ3n3D8E3"

		return
	}

	//nolint:exhaustive
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

func buildResponse(openAPIDoc openapi3.T, rst reflect.Type) openapi3.Responses {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{Type: "object"},
	}

	if rst != nil {
		schema = createComponent(openAPIDoc, rst)
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
	errComp := createComponent(openAPIDoc, t)

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

	//nolint:exhaustive
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

					schema.Properties[jsonName] = prop

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

			if note, ok := f.Tag.Lookup("note"); ok {
				p.Description = note
			}

			if validate, ok := f.Tag.Lookup("validate"); ok {
				for _, val := range strings.Split(validate, ",") {
					if val == "required" {
						p.Required = true
					}

					if strings.HasPrefix(val, "min=") {
						p.Schema.Value.MinLength = parseMinLength(val)
					}

					if strings.HasPrefix(val, "oneof=") {
						schema.Enum = parseOneOf(val)
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
		return "i:4yJ3n3D8E3"
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

func parseMinLength(tag string) uint64 {
	minLength := strings.Split(tag, "min=")
	if len(minLength) != 2 {
		panic("min length tag does not match expected format")
	}

	len, err := strconv.ParseUint(minLength[1], 10, 64)
	if err != nil {
		panic("unexpected min length: " + err.Error())
	}

	return len
}

func parseOneOf(tag string) []interface{} {
	oneof := strings.Split(tag, "oneof=")
	if len(oneof) != 2 {
		panic("oneof tag does not match expected format")
	}

	values := strings.Split(oneof[1], " ")

	// convert to a slice of interfaces to assign to the schema
	enumInterface := make([]interface{}, len(values))
	for i := range values {
		enumInterface[i] = values[i]
	}

	return enumInterface
}
