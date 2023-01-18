package access

import (
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(rCtx RequestContext, destination *models.Destination) error {
	roles := []string{models.InfraAdminRole, models.InfraConnectorRole}
	if err := IsAuthorized(rCtx, roles...); err != nil {
		return HandleAuthErr(err, "destination", "create", roles...)
	}

	return data.CreateDestination(rCtx.DBTxn, destination)
}

func UpdateDestination(rCtx RequestContext, destination *models.Destination) error {
	roles := []string{models.InfraAdminRole, models.InfraConnectorRole}
	if err := IsAuthorized(rCtx, roles...); err != nil {
		return HandleAuthErr(err, "destination", "update", roles...)
	}

	return data.UpdateDestination(rCtx.DBTxn, destination)
}

func DeleteDestination(rCtx RequestContext, id uid.ID) error {
	if err := IsAuthorized(rCtx, models.InfraAdminRole); err != nil {
		return HandleAuthErr(err, "destination", "delete", models.InfraAdminRole)
	}

	return data.DeleteDestination(rCtx.DBTxn, id)
}
