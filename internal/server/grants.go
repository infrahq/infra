package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type ListGrantsResponse api.ListResponse[api.Grant]

func (r ListGrantsResponse) SetHeaders(h http.Header) {
	if r.LastUpdateIndex.Index > 0 {
		h.Set("Last-Update-Index", strconv.FormatInt(r.LastUpdateIndex.Index, 10))
	}
}

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) (*ListGrantsResponse, error) {
	rCtx := getRequestContext(c)

	rCtx.Response.AddLogFields(func(event *zerolog.Event) {
		event.Int64("lastUpdateIndex", r.LastUpdateIndex)
	})

	var subject uid.PolymorphicID
	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	var p data.Pagination
	opts := data.ListGrantsOptions{
		ByResource:                 r.Resource,
		BySubject:                  subject,
		ByDestination:              r.Destination,
		ExcludeConnectorGrant:      !r.ShowSystem,
		IncludeInheritedFromGroups: r.ShowInherited,
	}
	if r.Privilege != "" {
		opts.ByPrivileges = []string{r.Privilege}
	}
	if !r.IsBlockingRequest() {
		p = PaginationFromRequest(r.PaginationRequest)
		opts.Pagination = &p
	}

	grants, err := access.ListGrants(c, opts, r.LastUpdateIndex)
	if err != nil {
		return nil, err
	}

	rCtx.Response.AddLogFields(func(event *zerolog.Event) {
		event.Int("numGrants", len(grants.Grants))
	})

	result := api.NewListResponse(grants.Grants, PaginationToResponse(p), func(grant models.Grant) api.Grant {
		return *grant.ToAPI()
	})
	result.LastUpdateIndex.Index = grants.MaxUpdateIndex

	return (*ListGrantsResponse)(result), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.GrantRequest) (*api.CreateGrantResponse, error) {
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
		opts := data.ListGrantsOptions{
			ByResource:   grant.Resource,
			BySubject:    grant.Subject,
			ByPrivileges: []string{grant.Privilege},
		}
		grants, err := access.ListGrants(c, opts, 0)
		if err != nil {
			return nil, err
		}

		if len(grants.Grants) == 0 {
			return nil, fmt.Errorf("duplicate grant exists, but cannot be found")
		}

		return &api.CreateGrantResponse{Grant: grants.Grants[0].ToAPI()}, nil
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
		opts := data.ListGrantsOptions{
			ByResource:   access.ResourceInfraAPI,
			ByPrivileges: []string{models.InfraAdminRole},
		}
		infraAdminGrants, err := access.ListGrants(c, opts, 0)
		if err != nil {
			return nil, err
		}

		if len(infraAdminGrants.Grants) == 1 {
			return nil, fmt.Errorf("%w: cannot remove the last infra admin", internal.ErrBadRequest)
		}
	}

	return nil, access.DeleteGrant(c, r.ID)
}

func (a *API) UpdateGrants(c *gin.Context, r *api.UpdateGrantsRequest) (*api.EmptyResponse, error) {
	iden := access.GetRequestContext(c).Authenticated.User
	var addGrants []*models.Grant
	for _, g := range r.GrantsToAdd {
		grant, err := getGrantFromGrantRequest(c, g)
		if err != nil {
			return nil, err
		}
		grant.CreatedBy = iden.ID
		addGrants = append(addGrants, grant)
	}

	var rmGrants []*models.Grant
	for _, g := range r.GrantsToRemove {
		grant, err := getGrantFromGrantRequest(c, g)
		if err != nil {
			return nil, err
		}
		rmGrants = append(rmGrants, grant)
	}

	return nil, access.UpdateGrants(c, addGrants, rmGrants)
}

func getGrantFromGrantRequest(c *gin.Context, r api.GrantRequest) (*models.Grant, error) {
	var subject uid.PolymorphicID

	switch {
	case r.UserName != "":
		// lookup user name
		identity, err := access.GetIdentity(c, data.GetIdentityOptions{ByName: r.UserName})
		if err != nil {
			if errors.Is(err, internal.ErrNotFound) {
				return nil, fmt.Errorf("%w: couldn't find userName '%s'", internal.ErrBadRequest, r.UserName)
			}
			return nil, err
		}
		subject = uid.NewIdentityPolymorphicID(identity.ID)
	case r.GroupName != "":
		group, err := access.GetGroup(c, data.GetGroupOptions{ByName: r.GroupName})
		if err != nil {
			if errors.Is(err, internal.ErrNotFound) {
				return nil, fmt.Errorf("%w: couldn't find groupName '%s'", internal.ErrBadRequest, r.GroupName)
			}
			return nil, err
		}
		subject = uid.NewIdentityPolymorphicID(group.ID)
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	switch {
	case subject == "":
		return nil, fmt.Errorf("%w: must specify userName, user, or group", internal.ErrBadRequest)
	case r.Resource == "":
		return nil, fmt.Errorf("%w: must specify resource", internal.ErrBadRequest)
	case r.Privilege == "":
		return nil, fmt.Errorf("%w: must specify privilege", internal.ErrBadRequest)
	}

	return &models.Grant{
		Subject:   subject,
		Resource:  r.Resource,
		Privilege: r.Privilege,
	}, nil
}
