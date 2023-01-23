package openapi3

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// Doc is the root of an OpenAPI v3 document
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#openapi-object
type Doc struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"` // Required
	Components openapi3.Components   `json:"components,omitempty" yaml:"components,omitempty"`
	Info       *Info                 `json:"info" yaml:"info"`   // Required
	Paths      map[string]*PathItem  `json:"paths" yaml:"paths"` // Required
	Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
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
	Responses   openapi3.Responses    `json:"responses" yaml:"responses"` // Required
	Callbacks   openapi3.Callbacks    `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security    []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers     []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
}

func (o *Operation) AddParameter(p *Parameter) {
	o.Parameters = append(o.Parameters, p)
}

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
	Schema   *openapi3.SchemaRef           `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}                   `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]*openapi3.Example  `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]*openapi3.Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// Parameter is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#parameter-object
type Parameter struct {
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	AllowReserved   bool                `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Description     string              `json:"description,omitempty" yaml:"description,omitempty"`
	Example         interface{}         `json:"example,omitempty" yaml:"example,omitempty"`
	Examples        openapi3.Examples   `json:"examples,omitempty" yaml:"examples,omitempty"`
	Explode         *bool               `json:"explode,omitempty" yaml:"explode,omitempty"`
	In              string              `json:"in,omitempty" yaml:"in,omitempty"`
	Name            string              `json:"name,omitempty" yaml:"name,omitempty"`
	Required        bool                `json:"required,omitempty" yaml:"required,omitempty"`
	Schema          *openapi3.SchemaRef `json:"schema,omitempty" yaml:"schema,omitempty"`
	Style           string              `json:"style,omitempty" yaml:"style,omitempty"`
	Content         openapi3.Content    `json:"content,omitempty" yaml:"content,omitempty"`
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
