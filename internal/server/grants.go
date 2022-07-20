package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
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

	// check if grant already exists
	grants, err := access.ListGrants(c, grant.Subject, grant.Resource, grant.Privilege, false, &models.Pagination{})
	if err != nil {
		return nil, err
	}

	if len(grants) == 1 {
		// this grant already exists
		return &api.CreateGrantResponse{Grant: grants[0].ToAPI()}, nil
	} else if len(grants) > 1 {
		return nil, fmt.Errorf("multiple duplicate grants exists")
	}

	err = access.CreateGrant(c, grant)
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
