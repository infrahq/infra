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
	"context"
	"errors"
	"net/http"
)

// SourcesApiService is a service that implents the logic for the SourcesApiServicer
// This service should implement the business logic for every endpoint for the SourcesApi API.
// Include any external packages or services that will be required by this service.
type SourcesApiService struct {
}

// NewSourcesApiService creates a default api service
func NewSourcesApiService() SourcesApiServicer {
	return &SourcesApiService{}
}

// ListSources - List sources
func (s *SourcesApiService) ListSources(ctx context.Context) (ImplResponse, error) {
	// TODO - update ListSources with the required logic for this service method.
	// Add api_sources_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, []Source{}) or use other options such as http.Ok ...
	//return Response(200, []Source{}), nil

	//TODO: Uncomment the next line to return response Response(0, Error{}) or use other options such as http.Ok ...
	//return Response(0, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("ListSources method not implemented")
}
