package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	destinations, err := access.ListDestinations(c, r.UniqueID, r.Name, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(destinations, models.PaginationToResponse(p), func(destination models.Destination) api.Destination {
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
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
		Version:       r.Version,
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{
		Model: models.Model{
			ID: r.ID,
		},
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
		Version:       r.Version,
	}

	if err := access.SaveDestination(c, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteDestination(c, r.ID)
}
