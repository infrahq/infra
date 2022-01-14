package access

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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

func ListGrants(c *gin.Context, kind models.GrantKind, destinationID uid.ID) ([]models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db, data.ByGrantKind(kind), data.ByDestinationID(destinationID))
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

// TODO: #760 - needed to sync grants when a destination is registered or changed
func SyncGrants(c *gin.Context, sync func(db *gorm.DB) error) error {
	db, err := requireAuthorization(c)
	if err != nil {
		return err
	}

	return sync(db)
}
