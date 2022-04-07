package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(c *gin.Context, destination *models.Destination) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return err
	}

	return data.CreateDestination(db, destination)
}

func SaveDestination(c *gin.Context, destination *models.Destination) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return err
	}

	return data.SaveDestination(db, destination)
}

func GetDestination(c *gin.Context, id uid.ID) (*models.Destination, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole, models.InfraUserRole)
	if err != nil {
		return nil, err
	}

	return data.GetDestination(db, data.ByID(id))
}

func ListDestinations(c *gin.Context, uniqueID, name string) ([]models.Destination, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole, models.InfraUserRole)
	if err != nil {
		return nil, err
	}

	return data.ListDestinations(db, data.ByUniqueID(uniqueID), data.ByName(name))
}

func DeleteDestination(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteDestinations(db, data.ByID(id))
}
