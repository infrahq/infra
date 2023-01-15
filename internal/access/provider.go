package access

import (
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateProvider(rCtx RequestContext, provider *models.Provider) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "create", models.InfraAdminRole)
	}

	return data.CreateProvider(rCtx.DBTxn, provider)
}

func SaveProvider(rCtx RequestContext, provider *models.Provider) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "update", models.InfraAdminRole)
	}
	if data.InfraProvider(rCtx.DBTxn).ID == provider.ID {
		return fmt.Errorf("%w: the infra provider can not be modified", internal.ErrBadRequest)
	}

	return data.UpdateProvider(rCtx.DBTxn, provider)
}

func DeleteProvider(rCtx RequestContext, id uid.ID) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "delete", models.InfraAdminRole)
	}
	if data.InfraProvider(rCtx.DBTxn).ID == id {
		return fmt.Errorf("%w: the infra provider can not be deleted", internal.ErrBadRequest)
	}

	return data.DeleteProviders(rCtx.DBTxn, data.DeleteProvidersOptions{ByID: id})
}
