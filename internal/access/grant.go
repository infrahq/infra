package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id), data.NotCreatedBy(models.CreatedBySystem))
}

func ListGrants(c *gin.Context, subject uid.PolymorphicID, resource string, privilege string) ([]models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db, data.BySubject(subject), data.ByResource(resource), data.ByPrivilege(privilege), data.NotCreatedBy(models.CreatedBySystem))
}

func ListUserGrants(c *gin.Context, userID uid.ID) ([]models.Grant, error) {
	db, err := hasAuthorization(c, userID, isUserSelf, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListUserGrants(db, userID)
}

func ListMachineGrants(c *gin.Context, machineID uid.ID) ([]models.Grant, error) {
	db, err := hasAuthorization(c, machineID, isMachineSelf, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListMachineGrants(db, machineID)
}

func ListGroupGrants(c *gin.Context, groupID uid.ID) ([]models.Grant, error) {
	db, err := hasAuthorization(c, groupID, isUserInGroup, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListGroupGrants(db, groupID)
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	creator := getCurrentIdentity(c)

	creatorID, err := creator.ID()
	if err != nil {
		return fmt.Errorf("set id from context: %w", err)
	}

	grant.CreatedBy = creatorID

	return data.CreateGrant(db, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteGrants(db, data.ByID(id), data.NotCreatedBy(models.CreatedBySystem))
}
