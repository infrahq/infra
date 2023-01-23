package openapi3

import "github.com/getkin/kin-openapi/openapi3"

// Doc is the root of an OpenAPI v3 document
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#openapi-object
type Doc struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"` // Required
	Components openapi3.Components   `json:"components,omitempty" yaml:"components,omitempty"`
	Info       *Info                 `json:"info" yaml:"info"`   // Required
	Paths      openapi3.Paths        `json:"paths" yaml:"paths"` // Required
	Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
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
