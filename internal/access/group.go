package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionGroup       Permission = "infra.group.*"
	PermissionGroupCreate Permission = "infra.group.create"
	PermissionGroupRead   Permission = "infra.group.read"
	PermissionGroupUpdate Permission = "infra.group.update"
	PermissionGroupDelete Permission = "infra.group.delete"
)

func ListGroups(c *gin.Context, name string, providerID uid.ID) ([]models.Group, error) {
	db, err := requireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(db, data.ByName(name), data.ByProviderID(providerID))
}

func CreateGroup(c *gin.Context, group *models.Group) error {
	db, err := requireAuthorization(c, PermissionGroupCreate)
	if err != nil {
		return err
	}

	return data.CreateGroup(db, group)
}

func GetGroup(c *gin.Context, id uid.ID) (*models.Group, error) {
	db, err := requireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.GetGroup(db, data.ByID(id))
}

func ListUserGroups(c *gin.Context, userID uid.ID) ([]models.Group, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionGroupRead, func(id uid.ID) bool {
		return userID == id
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserGroups(db, userID)
}
