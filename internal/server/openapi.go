package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/validate"
)

func GenerateOpenAPIDoc() openapi3.T {
	srv := newServer(Options{})
	srv.metricsRegistry = prometheus.NewRegistry()
	return srv.GenerateRoutes().OpenAPIDocument
}

var pathIDReplacer = regexp.MustCompile(`:\w+`)

// funcPartialNameToTagNames is a sorted (alphabetically by tag name) list of
// function name substrings to the tags associated with the operation.
var funcPartialNameToTagNames = []struct {
	partial string
	tag     string
}{
	{partial: "AccessKey", tag: "Authentication"},
	{partial: "Login", tag: "Authentication"},
	{partial: "Logout", tag: "Authentication"},
	{partial: "Destination", tag: "Destinations"},
	{partial: "Token", tag: "Destinations"},
	{partial: "Grant", tag: "Grants"},
	{partial: "Group", tag: "Groups"},
	{partial: "Provider", tag: "Providers"},
	{partial: "User", tag: "Users"},
}

// openAPIRouteDefinition converts the route into a format that can be used
// by API.register. This is necessary because currently methods can not have
// generic parameters.
func openAPIRouteDefinition[Req, Res any](route route[Req, Res]) (
	method string,
	path string,
	funcName string,
	requestType reflect.Type,
	resultType reflect.Type,
) {
	//nolint:gocritic
	reqT, resultT := reflect.TypeOf(*new(Req)), reflect.TypeOf(*new(Res))
	return route.method, route.path, getFuncName(route.handler), reqT, resultT
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
	buildRequest(rqt, op, method)
	op.Responses = buildResponse(a.openAPIDoc.Components.Schemas, rst)

	for _, item := range funcPartialNameToTagNames {
		if strings.Contains(funcName, item.partial) {
			op.Tags = append(op.Tags, item.tag)
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

// createComponent creates and returns the SchemaRef for a response type.
func createComponent(schemas openapi3.Schemas, rst reflect.Type) *openapi3.SchemaRef {
	if rst.Kind() == reflect.Pointer {
		rst = rst.Elem()
	}
	if rst.Kind() != reflect.Struct {
		panic(fmt.Sprintf("openapi: unexpected kind %v (%v) for response struct", rst.Kind(), rst))
	}

	schema := &openapi3.Schema{
		Properties: openapi3.Schemas{},
	}

	// Reformat the name of generic types
	name := strings.ReplaceAll(rst.Name(), rst.PkgPath()+".", "")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "")

	for i := 0; i < rst.NumField(); i++ {
		f := rst.Field(i)

		if f.Tag.Get("json") == "-" {
			continue
		}

		if f.Anonymous {
			typeOrElem := f.Type
			if f.Type.Kind() == reflect.Pointer {
				typeOrElem = f.Type.Elem()
			}

			if typeOrElem.Kind() == reflect.Struct {
				for j := 0; j < typeOrElem.NumField(); j++ {
					af := typeOrElem.Field(j)
					schema.Properties[getFieldName(af, typeOrElem)] = buildProperty(af, af.Type, typeOrElem, schema)
				}
				continue
			}
		}
		schema.Properties[getFieldName(f, rst)] = buildProperty(f, f.Type, rst, schema)
	}

	if _, ok := schemas[name]; ok {
		return &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + name,
		}
	}

	schemas[name] = &openapi3.SchemaRef{Value: schema}
	return &openapi3.SchemaRef{
		Ref: "#/components/schemas/" + name,
	}
}

func buildProperty(f reflect.StructField, t, parent reflect.Type, parentSchema *openapi3.Schema) *openapi3.SchemaRef {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	s := &openapi3.Schema{}
	updateSchemaFromStructTags(f, s)
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

		if req, ok := reflect.New(t).Interface().(validate.Request); ok {
			for _, rule := range req.ValidationRules() {
				rule.DescribeSchema(s)
			}
		}
	}

	return &openapi3.SchemaRef{Value: s}
}

func writeOpenAPISpec(spec openapi3.T, version string, out io.Writer) error {
	spec.OpenAPI = "3.0.0"
	spec.Info = &openapi3.Info{
		Title:       "Infra API",
		Version:     version,
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

func WriteOpenAPIDocToFile(openAPIDoc openapi3.T, version string, filename string) error {
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := writeOpenAPISpec(openAPIDoc, version, fh); err != nil {
		return err
	}
	return nil
}

func updateSchemaFromStructTags(field reflect.StructField, schema *openapi3.Schema) {
	if example, ok := field.Tag.Lookup("example"); ok {
		schema.Example = example
	}

	if note, ok := field.Tag.Lookup("note"); ok {
		schema.Description = note
	}
}

type describeSchema interface {
	DescribeSchema(schema *openapi3.Schema)
}

// `type` can be one of the following only: "object", "array", "string", "number", "integer", "boolean", "null".
// `format` has a few defined types, but can be anything. https://swagger.io/docs/specification/data-models/data-types/
func setTypeInfo(t reflect.Type, schema *openapi3.Schema) {
	// TODO: convert to value earlier?
	value := reflect.New(t).Interface()
	if ds, ok := value.(describeSchema); ok {
		ds.DescribeSchema(schema)
		return
	}

	switch value.(type) {
	case time.Time:
		panic("field must use api.Time")
	case time.Duration:
		panic("field must use api.Duration")
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	//nolint:exhaustive
	switch t.Kind() {
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
		if t.Elem().Kind() == reflect.Uint8 {
			schema.Type = "string" // []byte
			schema.Format = "base64"
			return
		}
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

func buildResponse(schemas openapi3.Schemas, rst reflect.Type) openapi3.Responses {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{Type: "object"},
	}

	if rst != nil {
		schema = createComponent(schemas, rst)
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

	content := openapi3.Content{"application/json": &openapi3.MediaType{
		Schema: createComponent(schemas, reflect.TypeOf(api.Error{})),
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

// productVersion is a shim for testing. It allows us to patch the variable
// so that tests expect a consistent value that does not change with every release.
var productVersion = internal.FullVersion

func buildRequest(r reflect.Type, op *openapi3.Operation, method string) {
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}

	if r.Kind() != reflect.Struct {
		panic(fmt.Sprintf("openapi: unexpected kind %v (%v) for %v request struct", r.Kind(), r, op.OperationID))
	}

	op.Parameters = openapi3.NewParameters()

	op.AddParameter(&openapi3.Parameter{
		Name:     "Infra-Version",
		In:       "header",
		Required: true,
		Schema: &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Example:     productVersion(),
				Format:      `\d+\.\d+\(.\d+)?(-.\w(+\w)?)?`,
				Type:        "string",
				Description: "Version of the API being requested",
			},
		},
	})

	schema := &openapi3.Schema{
		Type:       "object",
		Properties: openapi3.Schemas{},
	}

	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			tmpOp := openapi3.NewOperation()

			buildRequest(f.Type, tmpOp, method)
			for _, param := range tmpOp.Parameters {
				if param.Value.Name != "Infra-Version" {
					op.AddParameter(param.Value)
				}
			}

			if req, ok := reflect.New(f.Type).Interface().(validate.Request); ok {
				for _, rule := range req.ValidationRules() {
					rule.DescribeSchema(schema)
				}
			}
			continue
		}

		propName := getFieldName(f, r)
		if propName == "" { // ignored field
			continue
		}
		propSchema := buildProperty(f, f.Type, r, schema)
		// Store all property schemas so that validation rules can update them.
		schema.Properties[propName] = propSchema
		if name, ok := f.Tag.Lookup("json"); ok && !strings.HasPrefix(name, "-") {
			continue
		}

		p := &openapi3.Parameter{Name: propName, Schema: propSchema}

		if _, ok := f.Tag.Lookup("form"); ok {
			p.In = "query"
		}
		if _, ok := f.Tag.Lookup("uri"); ok {
			p.In = "path"
			p.Required = true
		}

		if p.In == "" {
			// field isn't properly labelled
			panic(fmt.Sprintf("field %q of struct %q must have a tag (json, form, or uri) with a name or '-'", f.Name, r.Name()))
		}

		// TODO: share this with updateSchemaFromStructTags
		if example, ok := f.Tag.Lookup("example"); ok {
			p.Example = example
		}
		if note, ok := f.Tag.Lookup("note"); ok {
			p.Description = note
		}

		op.AddParameter(p)
	}

	if req, ok := reflect.New(r).Interface().(validate.Request); ok {
		for _, rule := range req.ValidationRules() {
			rule.DescribeSchema(schema)
		}
	}

	if len(schema.Properties) == 0 {
		return
	}

	// Remove any non-body parameter from the parent schema now that the validation
	// rules have had a chance to update them.
	for _, param := range op.Parameters {
		if param.Value.In != "" {
			delete(schema.Properties, param.Value.Name)
			schema.Required = removeString(schema.Required, param.Value.Name)
		}
	}

	switch method {
	// These methods accept arguments from a request body
	case http.MethodPut, http.MethodPost, http.MethodPatch:
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

	// Other methods accept arguments from the query and path
	default:
	}
}

func removeString(seq []string, name string) []string {
	for i, v := range seq {
		if v == name {
			return append(seq[:i], seq[i+1:]...)
		}
	}
	return seq
}

func getFieldName(f reflect.StructField, parent reflect.Type) string {
	if name, ok := f.Tag.Lookup("form"); ok {
		return name
	}

	if name, ok := f.Tag.Lookup("uri"); ok {
		return name
	}

	// lookup json tag last, as a field may have a uri or form name, but a
	// json name of "-".
	if name, ok := f.Tag.Lookup("json"); ok {
		name = strings.Split(name, ",")[0]
		if name == "-" {
			return ""
		}
		return name
	}

	panic(fmt.Sprintf("field %q of struct %q must have a tag (json, form, or uri) with a name or '-'", f.Name, parent.Name()))
}
