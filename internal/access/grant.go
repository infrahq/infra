package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionGrant       Permission = "infra.grant.*"
	PermissionGrantCreate Permission = "infra.grant.create"
	PermissionGrantRead   Permission = "infra.grant.read"
	PermissionGrantUpdate Permission = "infra.grant.update"
	PermissionGrantDelete Permission = "infra.grant.delete"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id))
}

func ListUserGrants(c *gin.Context, userID uid.ID) ([]models.Grant, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionGrantRead, func(user *models.User) bool {
		return userID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserGrants(db, userID)
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	db, err := requireAuthorization(c, PermissionGrantCreate)
	if err != nil {
		return err
	}

	return data.CreateGrant(db, grant)
}
