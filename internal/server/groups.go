package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListGroups(c *gin.Context, r *api.ListGroupsRequest) (*api.ListResponse[api.Group], error) {
	p := PaginationFromRequest(r.PaginationRequest)
	groups, err := access.ListGroups(c, r.Name, r.UserID, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(groups, PaginationToResponse(p), func(group models.Group) api.Group {
		return *group.ToAPI()
	})

	return result, nil
}

func (a *API) GetGroup(c *gin.Context, r *api.Resource) (*api.Group, error) {
	group, err := access.GetGroup(c, data.GetGroupOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) CreateGroup(c *gin.Context, r *api.CreateGroupRequest) (*api.Group, error) {
	group := &models.Group{
		Name: r.Name,
	}

	authIdent := getRequestContext(c).Authenticated.User
	if authIdent != nil {
		group.CreatedBy = authIdent.ID
	}

	err := access.CreateGroup(c, group)
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) DeleteGroup(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteGroup(c, r.ID)
}

func (a *API) UpdateUsersInGroup(c *gin.Context, r *api.UpdateUsersInGroupRequest) (*api.EmptyResponse, error) {
	return nil, access.UpdateUsersInGroup(c, r.ID, r.UserIDsToAdd, r.UserIDsToRemove)
}
