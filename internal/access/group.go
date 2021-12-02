package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionGroup       Permission = "infra.group.*"
	PermissionGroupCreate Permission = "infra.group.create"
	PermissionGroupRead   Permission = "infra.group.read"
	PermissionGroupUpdate Permission = "infra.group.update"
	PermissionGroupDelete Permission = "infra.group.delete"
)

func GetGroup(c *gin.Context, id string) (*models.Group, error) {
	db, _, err := RequireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	group, err := models.NewGroup(id)
	if err != nil {
		return nil, err
	}

	return data.GetGroup(data.GroupAssociations(db), group)
}

func ListGroups(c *gin.Context, name string) ([]models.Group, error) {
	db, _, err := RequireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(data.GroupAssociations(db), &models.Group{Name: name})
}
