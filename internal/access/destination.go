package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(c *gin.Context, destination *models.Destination) error {
	roles := []string{models.InfraAdminRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return HandleAuthErr(err, "destination", "create", roles...)
	}

	return data.CreateDestination(db, destination)
}

func UpdateDestination(rCtx RequestContext, destination *models.Destination) error {
	roles := []string{models.InfraAdminRole, models.InfraConnectorRole}
	if err := IsAuthorized(rCtx, roles...); err != nil {
		return HandleAuthErr(err, "destination", "update", roles...)
	}

	return data.UpdateDestination(rCtx.DBTxn, destination)
}

func DeleteDestination(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "destination", "delete", models.InfraAdminRole)
	}

	return data.DeleteDestination(db, id)
}
