package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) (*api.ListResponse[api.Grant], error) {
	rCtx := getRequestContext(c)

	rCtx.Response.AddLogFields(func(event *zerolog.Event) {
		event.Int64("lastUpdateIndex", r.LastUpdateIndex)
	})

	var subject models.Subject
	switch {
	case r.User != 0:
		subject = models.NewSubjectForUser(r.User)
	case r.Group != 0:
		subject = models.NewSubjectForGroup(r.Group)
	}

	var p data.Pagination
	opts := data.ListGrantsOptions{
		ByDestinationName:          r.DestinationName,
		ByDestinationResource:      r.DestinationResource,
		BySubject:                  subject,
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

	grants, err := access.ListGrants(rCtx, opts, r.LastUpdateIndex)
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

	return result, nil
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	rCtx := getRequestContext(c)
	grant, err := access.GetGrant(rCtx, r.ID)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.GrantRequest) (*api.CreateGrantResponse, error) {
	rCtx := getRequestContext(c)
	grant, err := getGrantFromGrantRequest(rCtx, *r)
	if err != nil {
		return nil, err
	}

	err = access.CreateGrant(rCtx, grant)
	var ucerr data.UniqueConstraintError

	if errors.As(err, &ucerr) {
		opts := data.ListGrantsOptions{
			ByDestinationName:     grant.DestinationName,
			ByDestinationResource: grant.DestinationResource,
			BySubject:             grant.Subject,
			ByPrivileges:          []string{grant.Privilege},
		}
		grants, err := access.ListGrants(rCtx, opts, 0)

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
	rCtx := getRequestContext(c)
	grant, err := access.GetGrant(rCtx, r.ID)
	if err != nil {
		return nil, err
	}

	if grant.DestinationName == models.GrantDestinationInfra && grant.Privilege == models.InfraAdminRole {
		opts := data.ListGrantsOptions{
			ByDestinationName: models.GrantDestinationInfra,
			ByPrivileges:      []string{models.InfraAdminRole},
		}
		infraAdminGrants, err := access.ListGrants(rCtx, opts, 0)
		if err != nil {
			return nil, err
		}

		if len(infraAdminGrants.Grants) == 1 {
			return nil, fmt.Errorf("%w: cannot remove the last infra admin", internal.ErrBadRequest)
		}
	}

	return nil, access.DeleteGrant(rCtx, r.ID)
}

func (a *API) UpdateGrants(c *gin.Context, r *api.UpdateGrantsRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)
	iden := rCtx.Authenticated.User
	var addGrants []*models.Grant
	for _, g := range r.GrantsToAdd {
		grant, err := getGrantFromGrantRequest(rCtx, g)
		if err != nil {
			return nil, err
		}
		grant.CreatedBy = iden.ID
		addGrants = append(addGrants, grant)
	}

	var rmGrants []*models.Grant
	for _, g := range r.GrantsToRemove {
		grant, err := getGrantFromGrantRequest(rCtx, g)
		if err != nil {
			return nil, err
		}
		rmGrants = append(rmGrants, grant)
	}

	return nil, access.UpdateGrants(rCtx, addGrants, rmGrants)
}

func getGrantFromGrantRequest(rCtx access.RequestContext, r api.GrantRequest) (*models.Grant, error) {
	var subject models.Subject

	switch {
	case r.UserName != "":
		// lookup user name
		identity, err := access.GetIdentity(rCtx, data.GetIdentityOptions{ByName: r.UserName})
		if err != nil {
			if errors.Is(err, internal.ErrNotFound) {
				return nil, fmt.Errorf("%w: couldn't find userName '%s'", internal.ErrBadRequest, r.UserName)
			}
			return nil, err
		}
		subject = models.NewSubjectForUser(identity.ID)
	case r.GroupName != "":
		group, err := access.GetGroup(rCtx, data.GetGroupOptions{ByName: r.GroupName})
		if err != nil {
			if errors.Is(err, internal.ErrNotFound) {
				return nil, fmt.Errorf("%w: couldn't find groupName '%s'", internal.ErrBadRequest, r.GroupName)
			}
			return nil, err
		}
		subject = models.NewSubjectForGroup(group.ID)
	case r.User != 0:
		subject = models.NewSubjectForUser(r.User)
	case r.Group != 0:
		subject = models.NewSubjectForGroup(r.Group)
	}

	switch {
	case subject.ID == 0 || subject.Kind == 0:
		return nil, fmt.Errorf("%w: must specify userName, user, or group", internal.ErrBadRequest)
	case r.Privilege == "":
		return nil, fmt.Errorf("%w: must specify privilege", internal.ErrBadRequest)
	case r.DestinationName == "":
		return nil, fmt.Errorf("%w: must specify destination name", internal.ErrBadRequest)
	}

	return &models.Grant{
		Subject:             subject,
		Privilege:           r.Privilege,
		DestinationName:     r.DestinationName,
		DestinationResource: r.DestinationResource,
	}, nil
}

// See docs/dev/api-versioned-handlers.md for a guide to adding new version handlers.
func (a *API) addPreviousVersionHandlersGrants() {
	type grantV0_18_1 struct {
		ID        uid.ID   `json:"id"`
		Created   api.Time `json:"created"`
		CreatedBy uid.ID   `json:"created_by"`
		Updated   api.Time `json:"updated"`
		User      uid.ID   `json:"user,omitempty"`
		Group     uid.ID   `json:"group,omitempty"`
		Privilege string   `json:"privilege"`
		Resource  string   `json:"resource"`
	}

	newGrantsV0_18_1FromLatest := func(latest *api.Grant) *grantV0_18_1 {
		if latest == nil {
			return nil
		}

		return &grantV0_18_1{
			ID:        latest.ID,
			Created:   latest.Created,
			CreatedBy: latest.CreatedBy,
			Updated:   latest.Updated,
			User:      latest.User,
			Group:     latest.Group,
			Privilege: latest.Privilege,
			Resource:  api.FormatResourceURN(latest.DestinationName, latest.DestinationResource),
		}
	}

	type grantV0_21_0 grantV0_18_1

	newGrantV0_21_0FromLatest := func(latest *api.Grant) *grantV0_21_0 {
		if latest == nil {
			return nil
		}

		return &grantV0_21_0{
			ID:        latest.ID,
			Created:   latest.Created,
			CreatedBy: latest.CreatedBy,
			Updated:   latest.Updated,
			User:      latest.User,
			Group:     latest.Group,
			Privilege: latest.Privilege,
			Resource:  api.FormatResourceURN(latest.DestinationName, latest.DestinationResource),
		}
	}

	type listGrantsRequestV0_21_0 struct {
		User          uid.ID `form:"user"`
		Group         uid.ID `form:"group"`
		Resource      string `form:"resource"`
		Destination   string `form:"destination"`
		Privilege     string `form:"privilege"`
		ShowInherited bool   `form:"showInherited"`
		ShowSystem    bool   `form:"showSystem"`
		api.BlockingRequest
		api.PaginationRequest
	}

	newListGrantsRequestFromV0_21_0 := func(req *listGrantsRequestV0_21_0) (*api.ListGrantsRequest, error) {
		var destinationName, destinationResource string
		switch {
		case req.Destination != "":
			destinationName = req.Destination
		case req.Resource != "":
			destinationName, destinationResource = api.ParseResourceURN(req.Resource)
		}

		return &api.ListGrantsRequest{
			User:                req.User,
			Group:               req.Group,
			Privilege:           req.Privilege,
			DestinationName:     destinationName,
			DestinationResource: destinationResource,
			ShowInherited:       req.ShowInherited,
			ShowSystem:          req.ShowSystem,
			BlockingRequest:     req.BlockingRequest,
			PaginationRequest:   req.PaginationRequest,
		}, nil
	}

	// ListGrants
	addVersionHandler(a, http.MethodGet, "/api/grants", "0.18.1",
		route[api.ListGrantsRequest, *api.ListResponse[grantV0_18_1]]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(c *gin.Context, req *api.ListGrantsRequest) (*api.ListResponse[grantV0_18_1], error) {
				resp, err := a.ListGrants(c, req)
				return api.CopyListResponse(resp, func(item api.Grant) grantV0_18_1 {
					return *newGrantsV0_18_1FromLatest(&item)
				}), err
			},
		})

	addVersionHandler(a, http.MethodGet, "/api/grants", "0.21.0",
		route[listGrantsRequestV0_21_0, *api.ListResponse[grantV0_21_0]]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(c *gin.Context, req *listGrantsRequestV0_21_0) (*api.ListResponse[grantV0_21_0], error) {
				latest, err := newListGrantsRequestFromV0_21_0(req)
				if err != nil {
					return nil, err
				}

				resp, err := a.ListGrants(c, latest)
				if err != nil {
					return nil, err
				}

				return api.CopyListResponse(resp, func(item api.Grant) grantV0_21_0 {
					return *newGrantV0_21_0FromLatest(&item)
				}), err
			},
		})

	// GetGrants
	addVersionHandler(a, http.MethodGet, "/api/grants/:id", "0.18.1",
		route[api.Resource, *grantV0_18_1]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(c *gin.Context, req *api.Resource) (*grantV0_18_1, error) {
				resp, err := a.GetGrant(c, req)
				return newGrantsV0_18_1FromLatest(resp), err
			},
		})

	addVersionHandler(a, http.MethodGet, "/api/grants/:id", "0.21.0",
		route[api.Resource, *grantV0_21_0]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(c *gin.Context, req *api.Resource) (*grantV0_21_0, error) {
				resp, err := a.GetGrant(c, req)
				if err != nil {
					return nil, err
				}

				return newGrantV0_21_0FromLatest(resp), nil
			},
		},
	)

	type createGrantResponseV0_18_1 struct {
		*grantV0_18_1 `json:",inline"`
		WasCreated    bool `json:"wasCreated"`
	}

	type grantRequestV0_21_0 struct {
		User      uid.ID `json:"user"`
		Group     uid.ID `json:"group"`
		UserName  string `json:"userName"`
		GroupName string `json:"groupName"`
		Privilege string `json:"privilege"`
		Resource  string `json:"resource"`
	}

	newGrantRequestFromV0_21_0 := func(req *grantRequestV0_21_0) *api.GrantRequest {
		destinationName, destinationResource := api.ParseResourceURN(req.Resource)
		return &api.GrantRequest{
			User:                req.User,
			Group:               req.Group,
			UserName:            req.UserName,
			GroupName:           req.GroupName,
			Privilege:           req.Privilege,
			DestinationName:     destinationName,
			DestinationResource: destinationResource,
		}
	}

	type createGrantResponseV0_21_0 struct {
		*grantV0_21_0 `json:",inline"`
		WasCreated    bool `json:"-"`
	}

	// CreateGrant
	addVersionHandler(a, http.MethodPost, "/api/grants", "0.18.1",
		route[grantRequestV0_21_0, *createGrantResponseV0_18_1]{
			handler: func(c *gin.Context, req *grantRequestV0_21_0) (*createGrantResponseV0_18_1, error) {
				resp, err := a.CreateGrant(c, newGrantRequestFromV0_21_0(req))
				if err != nil {
					return nil, err
				}
				return &createGrantResponseV0_18_1{
					grantV0_18_1: newGrantsV0_18_1FromLatest(resp.Grant),
					WasCreated:   resp.WasCreated,
				}, nil
			},
		})

	addVersionHandler(a, http.MethodPost, "/api/grants", "0.21.0",
		route[grantRequestV0_21_0, *createGrantResponseV0_21_0]{
			handler: func(c *gin.Context, req *grantRequestV0_21_0) (*createGrantResponseV0_21_0, error) {
				resp, err := a.CreateGrant(c, newGrantRequestFromV0_21_0(req))
				if err != nil {
					return nil, err
				}

				return &createGrantResponseV0_21_0{
					grantV0_21_0: newGrantV0_21_0FromLatest(resp.Grant),
					WasCreated:   resp.WasCreated,
				}, nil
			},
		},
	)

	type updateGrantsRequestV0_21_0 struct {
		GrantsToAdd    []grantRequestV0_21_0 `json:"grantsToAdd"`
		GrantsToRemove []grantRequestV0_21_0 `json:"grantsToRemove"`
	}

	// UpdateGrants
	addVersionHandler(a, http.MethodPatch, "/api/grants", "0.21.0",
		route[updateGrantsRequestV0_21_0, *api.EmptyResponse]{
			handler: func(c *gin.Context, req *updateGrantsRequestV0_21_0) (*api.EmptyResponse, error) {
				var latest api.UpdateGrantsRequest
				for i := range req.GrantsToAdd {
					latest.GrantsToAdd = append(latest.GrantsToAdd, *newGrantRequestFromV0_21_0(&req.GrantsToAdd[i]))
				}

				for i := range req.GrantsToRemove {
					latest.GrantsToRemove = append(latest.GrantsToRemove, *newGrantRequestFromV0_21_0(&req.GrantsToRemove[i]))
				}

				resp, err := a.UpdateGrants(c, &latest)
				if err != nil {
					return nil, err
				}

				return resp, nil
			},
		},
	)
}
