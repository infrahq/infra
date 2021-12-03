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
