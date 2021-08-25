/*
 * Infra API
 *
 * Infra REST API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// CredsApiController binds http requests to an api service and writes the service results to the http response
type CredsApiController struct {
	service      CredsApiServicer
	errorHandler ErrorHandler
}

// CredsApiOption for how the controller is set up.
type CredsApiOption func(*CredsApiController)

// WithCredsApiErrorHandler inject ErrorHandler into controller
func WithCredsApiErrorHandler(h ErrorHandler) CredsApiOption {
	return func(c *CredsApiController) {
		c.errorHandler = h
	}
}

// NewCredsApiController creates a default api controller
func NewCredsApiController(s CredsApiServicer, opts ...CredsApiOption) Router {
	controller := &CredsApiController{
		service:      s,
		errorHandler: DefaultErrorHandler,
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

// Routes returns all of the api route for the CredsApiController
func (c *CredsApiController) Routes() Routes {
	return Routes{
		{
			"CreateCred",
			strings.ToUpper("Post"),
			"/creds",
			c.CreateCred,
		},
	}
}

// CreateCred - Create credentials to access a destination
func (c *CredsApiController) CreateCred(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.CreateCred(r.Context())
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}
