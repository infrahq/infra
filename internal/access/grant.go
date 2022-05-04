package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id))
}

func ListGrants(c *gin.Context, subject uid.PolymorphicID, resource string, privilege string) ([]models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db,
		data.ByOptionalSubject(subject),
		data.ByOptionalResource(resource),
		data.ByOptionalPrivilege(privilege))
}

func ListIdentityGrants(c *gin.Context, identityID uid.ID) ([]models.Grant, error) {
	db, err := hasAuthorization(c, identityID, isIdentitySelf, models.InfraAdminRole, models.InfraViewRole)
	if err != nil {
		return nil, err
	}

	return data.ListIdentityGrants(db, identityID)
}

func ListGroupGrants(c *gin.Context, groupID uid.ID) ([]models.Grant, error) {
	db, err := hasAuthorization(c, groupID, isUserInGroup, models.InfraAdminRole, models.InfraViewRole)
	if err != nil {
		return nil, err
	}

	return data.ListGroupGrants(db, groupID)
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	if err := checkRole(grant.Privilege, grant.Resource); err != nil {
		return err
	}

	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	creator := AuthenticatedIdentity(c)

	grant.CreatedBy = creator.ID

	return data.CreateGrant(db, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteGrants(db, data.ByID(id))
}

func checkRole(role, resource string) error {
	if resource == models.InternalInfraProviderName {
		switch role {
		case models.InfraAdminRole, models.InfraConnectorRole, models.InfraViewRole:
			return nil
		default:
			return fmt.Errorf("%w: [%s] is not a valid role for infra", internal.ErrBadRequest, role)
		}
	}
	return nil
}
