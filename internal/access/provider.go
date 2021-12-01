package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/data"
)

const (
	PermissionProvider       Permission = "infra.provider.*"
	PermissionProviderCreate Permission = "infra.provider.create"
	PermissionProviderRead   Permission = "infra.provider.read"
	PermissionProviderUpdate Permission = "infra.provider.update"
	PermissionProviderDelete Permission = "infra.provider.delete"
)

func GetProvider(c *gin.Context, id string) (*data.Provider, error) {
	db, _, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, err
	}

	provider, err := data.NewProvider(id)
	if err != nil {
		return nil, err
	}

	return data.GetProvider(db, provider)
}

func ListProviders(c *gin.Context, kind, domain string) ([]data.Provider, error) {
	db, _, err := RequireAuthorization(c, Permission(""))
	if err != nil {
		return nil, err
	}

	return data.ListProviders(db, &data.Provider{Kind: data.ProviderKind(kind), Domain: domain})
}
