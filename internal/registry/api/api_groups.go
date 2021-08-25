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

// GroupsApiController binds http requests to an api service and writes the service results to the http response
type GroupsApiController struct {
	service      GroupsApiServicer
	errorHandler ErrorHandler
}

// GroupsApiOption for how the controller is set up.
type GroupsApiOption func(*GroupsApiController)

// WithGroupsApiErrorHandler inject ErrorHandler into controller
func WithGroupsApiErrorHandler(h ErrorHandler) GroupsApiOption {
	return func(c *GroupsApiController) {
		c.errorHandler = h
	}
}

// NewGroupsApiController creates a default api controller
func NewGroupsApiController(s GroupsApiServicer, opts ...GroupsApiOption) Router {
	controller := &GroupsApiController{
		service:      s,
		errorHandler: DefaultErrorHandler,
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

// Routes returns all of the api route for the GroupsApiController
func (c *GroupsApiController) Routes() Routes {
	return Routes{
		{
			"ListGroups",
			strings.ToUpper("Get"),
			"/groups",
			c.ListGroups,
		},
	}
}

// ListGroups - List groups
func (c *GroupsApiController) ListGroups(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.ListGroups(r.Context())
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}
