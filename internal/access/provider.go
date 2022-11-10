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
	return data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{ByID: id})
}

func ListProviders(c *gin.Context, opts data.ListProvidersOptions) ([]models.Provider, error) {
	rCtx := GetRequestContext(c)
	return data.ListProviders(rCtx.DBTxn, opts)
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
