package access

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/uuid"

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

func GetGrant(c *gin.Context, id uuid.UUID) (*models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id))
}

func ListGrants(c *gin.Context, kind models.GrantKind, destinationID uuid.UUID) ([]models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db, data.ByGrantKind(kind), data.ByDestinationID(destinationID))
}

func ListUserGrants(c *gin.Context, userID uuid.UUID) ([]models.Grant, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionGrantRead, func(user *models.User) bool {
		return userID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserGrants(db, userID)
}
