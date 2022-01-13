package access

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

func GetGroup(c *gin.Context, id uuid.UUID) (*models.Group, error) {
	db, err := requireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.GetGroup(db, data.ByID(id))
}

func ListGroups(c *gin.Context, name string) ([]models.Group, error) {
	db, err := requireAuthorization(c, PermissionGroupRead)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(db, data.ByName(name))
}

func ListUserGroups(c *gin.Context, userID uuid.UUID) ([]models.Group, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionGroupRead, func(user *models.User) bool {
		return userID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserGroups(db, userID)
}
