package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type ListGrantsResponse api.ListResponse[api.Grant]

func (r ListGrantsResponse) Headers() http.Header {
	h := http.Header{}
	if r.LastUpdateIndex.Index > 0 {
		h.Set("Last-Update-Index", strconv.FormatInt(r.LastUpdateIndex.Index, 10))
	}
	return h
}

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) (*ListGrantsResponse, error) {
	var subject uid.PolymorphicID
	p := PaginationFromRequest(r.PaginationRequest)
	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	opts := data.ListGrantsOptions{
		ByResource:                 r.Resource,
		BySubject:                  subject,
		ExcludeConnectorGrant:      !r.ShowSystem,
		IncludeInheritedFromGroups: r.ShowInherited,
		Pagination:                 &p,
		IncludeMaxUpdateIndex:      r.LastUpdateIndex > 0,
	}
	if r.Privilege != "" {
		opts.ByPrivileges = []string{r.Privilege}
	}
	grants, err := access.ListGrants(c, opts, r.LastUpdateIndex)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(grants.Grants, PaginationToResponse(p), func(grant models.Grant) api.Grant {
		return *grant.ToAPI()
	})
	result.LastUpdateIndex.Index = grants.MaxUpdateIndex

	return (*ListGrantsResponse)(result), nil
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.CreateGrantRequest) (*api.CreateGrantResponse, error) {
	rCtx := getRequestContext(c)
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
		grant, err = data.GetGrant(rCtx.DBTxn, data.GetGrantOptions{
			BySubject:   grant.Subject,
			ByResource:  grant.Resource,
			ByPrivilege: grant.Privilege,
		})
		if err != nil {
			return nil, err
		}
		return &api.CreateGrantResponse{Grant: grant.ToAPI()}, nil
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
		infraAdminGrants, err := access.ListGrants(c, data.ListGrantsOptions{
			ByResource:   access.ResourceInfraAPI,
			ByPrivileges: []string{models.InfraAdminRole},
		}, 0)
		if err != nil {
			return nil, err
		}

		if len(infraAdminGrants.Grants) == 1 {
			return nil, fmt.Errorf("%w: cannot remove the last infra admin", internal.ErrBadRequest)
		}
	}

	return nil, access.DeleteGrant(c, r.ID)
}
