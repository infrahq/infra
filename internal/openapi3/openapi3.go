package openapi3

import "github.com/getkin/kin-openapi/openapi3"

// Doc is the root of an OpenAPI v3 document
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#openapi-object
type Doc struct {
	OpenAPI      string                        `json:"openapi" yaml:"openapi"` // Required
	Components   openapi3.Components           `json:"components,omitempty" yaml:"components,omitempty"`
	Info         *openapi3.Info                `json:"info" yaml:"info"`   // Required
	Paths        openapi3.Paths                `json:"paths" yaml:"paths"` // Required
	Security     openapi3.SecurityRequirements `json:"security,omitempty" yaml:"security,omitempty"`
	Servers      openapi3.Servers              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags         openapi3.Tags                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs *openapi3.ExternalDocs        `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}
