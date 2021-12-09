package access

import (
	"github.com/gin-gonic/gin"

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
	db, err := RequireAuthorization(c, PermissionProviderCreate)
	if err != nil {
		return nil, err
	}

	return data.CreateProvider(db, provider)
}

func GetProvider(c *gin.Context, id string) (*models.Provider, error) {
	db, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, err
	}

	provider, err := models.NewProvider(id)
	if err != nil {
		return nil, err
	}

	return data.GetProvider(db, provider)
}

func ListProviders(c *gin.Context, kind, domain string) ([]models.Provider, error) {
	db, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, err
	}

	return data.ListProviders(db, &models.Provider{Kind: models.ProviderKind(kind), Domain: domain})
}

func UpdateProvider(c *gin.Context, id string, provider *models.Provider) (*models.Provider, error) {
	db, err := RequireAuthorization(c, PermissionProviderUpdate)
	if err != nil {
		return nil, err
	}

	return data.UpdateProvider(db, provider, data.ByID(id))
}

func DeleteProvider(c *gin.Context, id string) error {
	db, err := RequireAuthorization(c, PermissionProviderDelete)
	if err != nil {
		return err
	}

	return data.DeleteProviders(db, data.ByID(id))
}
