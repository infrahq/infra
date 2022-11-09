package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "create", models.InfraAdminRole)
	}

	return data.CreateProvider(db, provider)
}

func GetProvider(c *gin.Context, id uid.ID) (*models.Provider, error) {
	rCtx := GetRequestContext(c)
	provider, err := data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{ByID: id})
	if err != nil {
		return nil, err
	}
	// if the caller is not authenticated for this org do not return the allowed domains, it is sensitive info
	reqKey := rCtx.Authenticated.AccessKey
	if reqKey == nil || reqKey.OrganizationID != rCtx.DBTxn.OrganizationID() {
		// sanitize the response, the caller is not authorized to know this
		provider.AllowedDomains = []string{}
	}
	return provider, nil
}

func ListProviders(c *gin.Context, opts data.ListProvidersOptions) ([]models.Provider, error) {
	rCtx := GetRequestContext(c)
	providers, err := data.ListProviders(rCtx.DBTxn, opts)
	if err != nil {
		return nil, err
	}
	// if the caller is not authenticated for this org do not return the allowed domains, it is sensitive info
	reqKey := rCtx.Authenticated.AccessKey
	if reqKey == nil || reqKey.OrganizationID != rCtx.DBTxn.OrganizationID() {
		// sanitize the response, the caller is not authorized to know this
		for i := range providers {
			providers[i].AllowedDomains = []string{}
		}
	}
	return providers, nil
}

func SaveProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "update", models.InfraAdminRole)
	}
	if data.InfraProvider(db).ID == provider.ID {
		return fmt.Errorf("%w: the infra provider can not be modified", internal.ErrBadRequest)
	}

	return data.UpdateProvider(db, provider)
}

func DeleteProvider(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "delete", models.InfraAdminRole)
	}
	if data.InfraProvider(db).ID == id {
		return fmt.Errorf("%w: the infra provider can not be deleted", internal.ErrBadRequest)
	}

	return data.DeleteProviders(db, data.DeleteProvidersOptions{ByID: id})
}

func GetSocialLoginProvider(c *gin.Context, kind models.ProviderKind) (*models.Provider, error) {
	rCtx := GetRequestContext(c)
	provider, err := data.GetSocialLoginProvider(rCtx.DBTxn, kind)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func ListSocialLoginProviders(c *gin.Context, p *data.Pagination) ([]models.Provider, error) {
	rCtx := GetRequestContext(c)
	providers, err := data.ListSocialLoginProviders(rCtx.DBTxn, p)
	if err != nil {
		return nil, err
	}
	return providers, nil
}
