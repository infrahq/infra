package access

import (
	"github.com/gin-gonic/gin"

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

func ListGrants(c *gin.Context, kind, destinationID string) ([]models.Grant, error) {
	db, err := RequireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	grant := models.Grant{
		Resource: models.Resource{
			Kind: kind,
		},
	}

	return data.ListGrants(db, &grant)
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
