package server

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListDestinations(rCtx access.RequestContext, r *api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	p := PaginationFromRequest(r.PaginationRequest)

	opts := data.ListDestinationsOptions{
		ByUniqueID: r.UniqueID,
		ByName:     r.Name,
		ByKind:     r.Kind,
		Pagination: &p,
	}
	destinations, err := data.ListDestinations(rCtx.DBTxn, opts)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(destinations, PaginationToResponse(p), func(destination models.Destination) api.Destination {
		return *destination.ToAPI()
	})

	return result, nil
}

func (a *API) GetDestination(rCtx access.RequestContext, r *api.Resource) (*api.Destination, error) {
	// No authorization required to view a destination
	destination, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(rCtx access.RequestContext, r *api.CreateDestinationRequest) (*api.Destination, error) {
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
	switch {
	case rCtx.Request.Header.Get(headerInfraDestinationName) == r.Name:
		destination.LastSeenAt = time.Now()
	case rCtx.Request.Header.Get(headerInfraDestinationUniqueID) == r.UniqueID:
		destination.LastSeenAt = time.Now()
	}

	err := access.CreateDestination(rCtx, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(rCtx access.RequestContext, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	// Start with the existing value, so that non-update fields are not set to zero.
	destination, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByID: r.ID})
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

func (a *API) DeleteDestination(rCtx access.RequestContext, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteDestination(rCtx, r.ID)
}
