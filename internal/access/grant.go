package access

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

func ListGrants(c *gin.Context, kind, destinationID string) ([]models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	grant := models.Grant{
		Kind: models.GrantKind(kind),
	}

	if id, err := uuid.Parse(destinationID); err == nil {
		grant.DestinationID = id
	}

	return data.ListGrants(db, data.GrantSelector(db, &grant))
}
