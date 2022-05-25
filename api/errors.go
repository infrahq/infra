package api

import (
	"fmt"
	"net/http"
	"strings"
)

// Error is used as the response body for failed HTTP requests. It is also
// the error returned by api.Client methods when the request fails.
type Error struct {
	// Method is the HTTP request method.
	Method string `json:"method"`
	// Path is the HTTP request path.
	Path string `json:"path"`
	// Code is the HTTP status of the response.
	Code int32 `json:"code"`
	// Message contains the full text of the failure as a single string. The
	// details of the failure may also be available in a structured representation
	// from one of the other fields on the Error struct.
	Message string `json:"message"`
	// FieldErrors contains a structured representation of any validation errors.
	FieldErrors []FieldError `json:"fieldErrors,omitempty"`
}

func (e Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%d %v", e.Code, strings.ToLower(http.StatusText(int(e.Code))))
	}
	return e.Message
}

type FieldError struct {
	FieldName string   `json:"fieldName"`
	Errors    []string `json:"errors"`
}
