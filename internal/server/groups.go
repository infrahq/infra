package server

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListGroups(rCtx access.RequestContext, r *api.ListGroupsRequest) (*api.ListResponse[api.Group], error) {
	p := PaginationFromRequest(r.PaginationRequest)
	groups, err := access.ListGroups(rCtx, r.Name, r.UserID, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(groups, PaginationToResponse(p), func(group models.Group) api.Group {
		return *group.ToAPI()
	})

	return result, nil
}

func (a *API) GetGroup(rCtx access.RequestContext, r *api.Resource) (*api.Group, error) {
	group, err := access.GetGroup(rCtx, data.GetGroupOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) CreateGroup(rCtx access.RequestContext, r *api.CreateGroupRequest) (*api.Group, error) {
	group := &models.Group{Name: r.Name}

	authIdent := rCtx.Authenticated.User
	if authIdent != nil {
		group.CreatedBy = authIdent.ID
	}

	err := access.CreateGroup(rCtx, group)
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) DeleteGroup(rCtx access.RequestContext, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteGroup(rCtx, r.ID)
}

func (a *API) UpdateUsersInGroup(rCtx access.RequestContext, r *api.UpdateUsersInGroupRequest) (*api.EmptyResponse, error) {
	return nil, access.UpdateUsersInGroup(rCtx, r.GroupID, r.UserIDsToAdd, r.UserIDsToRemove)
}
