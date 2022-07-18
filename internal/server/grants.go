package server

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) (*api.ListResponse[api.Grant], error) {
	var subject uid.PolymorphicID
	p := models.RequestToPagination(r.PaginationRequest)
	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	grants, err := access.ListGrants(c, subject, r.Resource, r.Privilege, r.ShowInherited, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(grants, models.PaginationToResponse(p), func(grant models.Grant) api.Grant {
		return *grant.ToAPI()
	})

	return result, nil
}

// TODO: remove after deprecation period
func (a *API) deprecatedListUserGrants(c *gin.Context, r *api.Resource) (*api.ListResponse[api.Grant], error) {
	return a.ListGrants(c, &api.ListGrantsRequest{User: r.ID})
}

// TODO: remove after deprecation period
func (a *API) deprecatedListGroupGrants(c *gin.Context, r *api.Resource) (*api.ListResponse[api.Grant], error) {
	return a.ListGrants(c, &api.ListGrantsRequest{Group: r.ID})
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.CreateGrantRequest) (*api.CreateGrantResponse, error) {
	var subject uid.PolymorphicID

	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	grant := &models.Grant{
		Subject:   subject,
		Resource:  r.Resource,
		Privilege: r.Privilege,
	}

	err := access.CreateGrant(c, grant)
	var ucerr data.UniqueConstraintError

	if errors.As(err, &ucerr) {
		grants, err := access.ListGrants(c, grant.Subject, grant.Resource, grant.Privilege, false, &models.Pagination{})

		if err != nil {
			return nil, err
		}

		if len(grants) == 0 {
			return nil, fmt.Errorf("duplicate grant exists, but cannot be found")
		}

		return &api.CreateGrantResponse{Grant: grants[0].ToAPI()}, nil
	}

	if err != nil {
		return nil, err
	}

	return &api.CreateGrantResponse{Grant: grant.ToAPI(), WasCreated: true}, nil

}

func (a *API) DeleteGrant(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	if grant.Resource == access.ResourceInfraAPI && grant.Privilege == models.InfraAdminRole {
		infraAdminGrants, err := access.ListGrants(c, "", grant.Resource, grant.Privilege, false, &models.Pagination{})
		if err != nil {
			return nil, err
		}

		if len(infraAdminGrants) == 1 {
			return nil, fmt.Errorf("%w: cannot remove the last infra admin", internal.ErrBadRequest)
		}
	}

	return nil, access.DeleteGrant(c, r.ID)
}
