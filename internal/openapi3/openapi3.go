package openapi3

// Doc is the root of an OpenAPI v3 document
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#openapi-object
type Doc struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"` // Required
	Components Components            `json:"components,omitempty" yaml:"components,omitempty"`
	Info       *Info                 `json:"info" yaml:"info"`   // Required
	Paths      map[string]*PathItem  `json:"paths" yaml:"paths"` // Required
	Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Components is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#components-object
type Components struct {
	Schemas    map[string]*SchemaRef `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Parameters map[string]*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Headers    map[string]*Parameter `json:"headers,omitempty" yaml:"headers,omitempty"`
	Responses  map[string]Response   `json:"responses,omitempty" yaml:"responses,omitempty"`
	Examples   map[string]Example    `json:"examples,omitempty" yaml:"examples,omitempty"`
	Callbacks  map[string]Callback   `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
}

// PathItem is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#path-item-object
type PathItem struct {
	Ref         string       `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Connect     *Operation   `json:"connect,omitempty" yaml:"connect,omitempty"`
	Delete      *Operation   `json:"delete,omitempty" yaml:"delete,omitempty"`
	Get         *Operation   `json:"get,omitempty" yaml:"get,omitempty"`
	Head        *Operation   `json:"head,omitempty" yaml:"head,omitempty"`
	Options     *Operation   `json:"options,omitempty" yaml:"options,omitempty"`
	Patch       *Operation   `json:"patch,omitempty" yaml:"patch,omitempty"`
	Post        *Operation   `json:"post,omitempty" yaml:"post,omitempty"`
	Put         *Operation   `json:"put,omitempty" yaml:"put,omitempty"`
	Trace       *Operation   `json:"trace,omitempty" yaml:"trace,omitempty"`
	Servers     []Server     `json:"servers,omitempty" yaml:"servers,omitempty"`
	Parameters  []*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Operation represents "operation" specified by" OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#operation-object
type Operation struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []*Parameter          `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses" yaml:"responses"` // Required
	Callbacks   map[string]Callback   `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security    []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers     []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
}

func (o *Operation) AddParameter(p *Parameter) {
	o.Parameters = append(o.Parameters, p)
}

type Callback map[string]*PathItem

// RequestBody is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#request-body-object
type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]*MediaType `json:"content" yaml:"content"`
}

// MediaType is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#media-type-object
type MediaType struct {
	Schema   *SchemaRef          `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}         `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]Example  `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// Parameter is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#parameter-object
type Parameter struct {
	AllowEmptyValue bool                  `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	AllowReserved   bool                  `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Deprecated      bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Description     string                `json:"description,omitempty" yaml:"description,omitempty"`
	Example         interface{}           `json:"example,omitempty" yaml:"example,omitempty"`
	Examples        map[string]Example    `json:"examples,omitempty" yaml:"examples,omitempty"`
	Explode         *bool                 `json:"explode,omitempty" yaml:"explode,omitempty"`
	In              string                `json:"in,omitempty" yaml:"in,omitempty"`
	Name            string                `json:"name,omitempty" yaml:"name,omitempty"`
	Required        bool                  `json:"required,omitempty" yaml:"required,omitempty"`
	Schema          *SchemaRef            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Style           string                `json:"style,omitempty" yaml:"style,omitempty"`
	Content         map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// SchemaRef represents either a Schema or a $ref to a Schema.
// When serializing and both fields are set, Ref is preferred over Value.
type SchemaRef struct {
	Ref     string `json:"$ref,omitempty"`
	*Schema `json:",inline,omitempty"`
}

// Schema is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#schema-object
type Schema struct {
	AdditionalProperties        *SchemaRef            `multijson:"additionalProperties,omitempty" json:"-" yaml:"-"`
	AdditionalPropertiesAllowed *bool                 `multijson:"additionalProperties,omitempty" json:"-" yaml:"-"`
	AllOf                       []*SchemaRef          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AllowEmptyValue             bool                  `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	AnyOf                       []*SchemaRef          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Default                     interface{}           `json:"default,omitempty" yaml:"default,omitempty"`
	Deprecated                  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Description                 string                `json:"description,omitempty" yaml:"description,omitempty"`
	Enum                        []interface{}         `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example                     interface{}           `json:"example,omitempty" yaml:"example,omitempty"`
	ExclusiveMax                bool                  `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	ExclusiveMin                bool                  `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	Format                      string                `json:"format,omitempty" yaml:"format,omitempty"`
	Items                       *SchemaRef            `json:"items,omitempty" yaml:"items,omitempty"`
	Max                         *float64              `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MaxItems                    *uint64               `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MaxLength                   *uint64               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MaxProps                    *uint64               `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Min                         *float64              `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MinItems                    uint64                `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MinLength                   uint64                `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MinProps                    uint64                `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MultipleOf                  *float64              `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Not                         *SchemaRef            `json:"not,omitempty" yaml:"not,omitempty"`
	Nullable                    bool                  `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	OneOf                       []*SchemaRef          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Pattern                     string                `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Properties                  map[string]*SchemaRef `json:"properties,omitempty" yaml:"properties,omitempty"`
	ReadOnly                    bool                  `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Required                    []string              `json:"required,omitempty" yaml:"required,omitempty"`
	Title                       string                `json:"title,omitempty" yaml:"title,omitempty"`
	Type                        string                `json:"type,omitempty" yaml:"type,omitempty"`
	UniqueItems                 bool                  `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	WriteOnly                   bool                  `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
}

// Example is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#example-object
type Example struct {
	Summary       string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string      `json:"description,omitempty" yaml:"description,omitempty"`
	Value         interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

// Encoding is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#encoding-object
type Encoding struct {
	ContentType   string                `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers       map[string]*Parameter `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style         string                `json:"style,omitempty" yaml:"style,omitempty"`
	Explode       *bool                 `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved bool                  `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
}

// Response is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#response-object
type Response struct {
	Content     map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Headers     map[string]*Parameter `json:"headers,omitempty" yaml:"headers,omitempty"`
	// Links are not used yet
}

// Info is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#info-object
type Info struct {
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Title          string   `json:"title" yaml:"title"`     // Required
	Version        string   `json:"version" yaml:"version"` // Required
}

// Contact is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#contact-object
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#license-object
type License struct {
	Name string `json:"name" yaml:"name"` // Required
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// SecurityRequirement is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#security-requirement-object
type SecurityRequirement map[string][]string

// Server is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#server-object
type Server struct {
	URL         string                     `json:"url" yaml:"url"`
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#server-variable-object
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}
