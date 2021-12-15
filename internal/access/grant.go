package access

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionGrant       Permission = "infra.grant.*"
	PermissionGrantCreate Permission = "infra.grant.create"
	PermissionGrantRead   Permission = "infra.grant.read"
	PermissionGrantUpdate Permission = "infra.grant.update"
	PermissionGrantDelete Permission = "infra.grant.delete"
)

func CreateGrant(c *gin.Context, grant *models.Grant) (*models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantCreate)
	if err != nil {
		return nil, err
	}

	return data.CreateGrant(db, grant)
}

func GetGrant(c *gin.Context, id string) (*models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	grant, err := models.NewGrant(id)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, grant)
}

func ListGrants(c *gin.Context, user *models.User, group *models.Group, destination *models.Destination) ([]models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	switch {
	case user != nil:
		return data.ListGrants(db, data.ByGrantUser(user))
	case group != nil:
		return data.ListGrants(db, data.ByGrantGroup(group))
	case destination != nil:
		return data.ListGrants(db, data.ByGrantDestination(destination))
	}

	return data.ListGrants(db, func(db *gorm.DB) *gorm.DB {
		return db
	})
}

func UpdateGrant(c *gin.Context, id string, grant *models.Grant) (*models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantUpdate)
	if err != nil {
		return nil, err
	}

	return data.UpdateGrant(db, grant, data.ByID(id))
}

func DeleteGrant(c *gin.Context, id string) error {
	db, err := RequireAuthorization(c, PermissionGrantDelete)
	if err != nil {
		return err
	}

	return data.DeleteGrants(db, data.ByID(id))
}
