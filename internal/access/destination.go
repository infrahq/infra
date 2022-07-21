package access

import (
	"fmt"

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

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}
	destination.OrganizationID = orgID

	return data.CreateDestination(db, destination)
}

func SaveDestination(c *gin.Context, destination *models.Destination) error {
	roles := []string{models.InfraAdminRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return HandleAuthErr(err, "destination", "update", roles...)
	}

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}
	destination.OrganizationID = orgID

	return data.SaveDestination(db, destination)
}

func GetDestination(c *gin.Context, id uid.ID) (*models.Destination, error) {
	db := getDB(c)

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get org for user")
	}

	return data.GetDestination(db, orgSelector, data.ByID(id))
}

func ListDestinations(c *gin.Context, uniqueID, name string, pg models.Pagination) ([]models.Destination, error) {
	db := getDB(c)

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get org for user")
	}

	return data.ListDestinations(db, orgSelector, data.ByOptionalUniqueID(uniqueID),
		data.ByOptionalName(name), data.ByPagination(pg))
}

func DeleteDestination(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "destination", "delete", models.InfraAdminRole)
	}

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return fmt.Errorf("Couldn't get org for user")
	}

	return data.DeleteDestinations(db, orgSelector, data.ByID(id))
}
