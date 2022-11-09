package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	p := PaginationFromRequest(r.PaginationRequest)

	opts := data.ListDestinationsOptions{
		ByUniqueID: r.UniqueID,
		ByName:     r.Name,
		ByKind:     r.Kind,
		Pagination: &p,
	}
	destinations, err := access.ListDestinations(c, opts)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(destinations, PaginationToResponse(p), func(destination models.Destination) api.Destination {
		return *destination.ToAPI()
	})

	return result, nil
}

func (a *API) GetDestination(c *gin.Context, r *api.Resource) (*api.Destination, error) {
	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		return nil, err
	}

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(c *gin.Context, r *api.CreateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		Kind:          models.DestinationKind(r.Kind),
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
		Version:       r.Version,
	}

	if destination.Kind == "" {
		destination.Kind = "kubernetes"
	}

	// set LastSeenAt if this request came from a connector. The middleware
	// can't do this update in the case where the destination did not exist yet
	if c.Request.Header.Get(headerInfraDestination) == r.UniqueID {
		destination.LastSeenAt = time.Now()
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	rCtx := getRequestContext(c)

	// Start with the existing value, so that non-update fields are not set to zero.
	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		return nil, err
	}

	destination.Name = r.Name
	destination.UniqueID = r.UniqueID
	destination.ConnectionURL = r.Connection.URL
	destination.ConnectionCA = string(r.Connection.CA)
	destination.Resources = r.Resources
	destination.Roles = r.Roles
	destination.Version = r.Version

	if err := access.UpdateDestination(rCtx, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteDestination(c, r.ID)
}
