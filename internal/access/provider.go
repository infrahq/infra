package access

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionProvider       Permission = "infra.provider.*"
	PermissionProviderCreate Permission = "infra.provider.create"
	PermissionProviderRead   Permission = "infra.provider.read"
	PermissionProviderUpdate Permission = "infra.provider.update"
	PermissionProviderDelete Permission = "infra.provider.delete"
)

func CreateProvider(c *gin.Context, provider *models.Provider) (*models.Provider, error) {
	db, err := requireAuthorization(c, PermissionProviderCreate)
	if err != nil {
		return nil, err
	}

	return data.CreateProvider(db, provider)
}

func GetProvider(c *gin.Context, id uuid.UUID) (*models.Provider, error) {
	db, err := requireAuthorization(c)
	if err != nil {
		return nil, err
	}

	return data.GetProvider(db, data.ByID(id))
}

func ListProviders(c *gin.Context, kind, domain string) ([]models.Provider, error) {
	db, err := requireAuthorization(c)
	if err != nil {
		return nil, err
	}

	return data.ListProviders(db, &models.Provider{Kind: models.ProviderKind(kind), Domain: domain})
}

func UpdateProvider(c *gin.Context, id uuid.UUID, provider *models.Provider) (*models.Provider, error) {
	db, err := requireAuthorization(c, PermissionProviderUpdate)
	if err != nil {
		return nil, err
	}

	return data.UpdateProvider(db, provider, data.ByID(id))
}

func DeleteProvider(c *gin.Context, id uuid.UUID) error {
	db, err := requireAuthorization(c, PermissionProviderDelete)
	if err != nil {
		return err
	}

	return data.DeleteProviders(db, data.ByID(id))
}
