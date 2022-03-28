package api

import (
	"fmt"

	"github.com/infrahq/infra/uid"
)

type Resource struct {
	ID uid.ID `uri:"id" validate:"required"`
}

var (
	// ErrUnauthorized refers to the http response code unauthorized, which really means not authenticated, despite its name. See https://stackoverflow.com/a/6937030/155585
	ErrUnauthorized = fmt.Errorf("unauthorized")
	// ErrForbidden means you don't have permissions to the requested resource
	ErrForbidden = fmt.Errorf("forbidden")
	// ErrBadGateway means an invalid response was received from an upstream server (probably an OIDC provider)
	ErrBadGateway = fmt.Errorf("bad gateway")

	ErrDuplicate  = fmt.Errorf("duplicate record")
	ErrNotFound   = fmt.Errorf("record not found")
	ErrBadRequest = fmt.Errorf("bad request")
	ErrInternal   = fmt.Errorf("internal error")
)

type Error struct {
	Code        int32        `json:"code"` // should be a repeat of the http response status code
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"fieldErrors,omitempty"`
}

type FieldError struct {
	FieldName string   `json:"fieldName"`
	Errors    []string `json:"errors"`
}
