package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/data"
)

const (
	PermissionGroup       Permission = "infra.group.*"
	PermissionGroupCreate Permission = "infra.group.create"
	PermissionGroupRead   Permission = "infra.group.read"
	PermissionGroupUpdate Permission = "infra.group.update"
	PermissionGroupDelete Permission = "infra.group.delete"
)

func GetGroup(c *gin.Context, id string) (*data.Group, error) {
	db, _, err := RequireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	group, err := data.NewGroup(id)
	if err != nil {
		return nil, err
	}

	return data.GetGroup(data.GroupAssociations(db), group)
}

func ListGroups(c *gin.Context, name string) ([]data.Group, error) {
	db, _, err := RequireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(data.GroupAssociations(db), &data.Group{Name: name})
}
