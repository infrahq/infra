package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isUserInGroup is used by authorization checks to see if the calling user is requesting their own attributes
func isUserInGroup(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	user := CurrentIdentity(c)

	if user != nil {
		lookupDB := getDB(c)

		groups, err := data.ListIdentityGroups(lookupDB, user.ID)
		if err != nil {
			return false, err
		}

		for _, g := range groups {
			if g.ID == requestedResourceID {
				return true, nil
			}
		}
	}

	return false, nil
}

func ListGroups(c *gin.Context, name string, providerID uid.ID) ([]models.Group, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListGroups(db, data.ByName(name), data.ByProviderID(providerID))
}

func CreateGroup(c *gin.Context, group *models.Group) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.CreateGroup(db, group)
}

func GetGroup(c *gin.Context, id uid.ID) (*models.Group, error) {
	db, err := hasAuthorization(c, id, isUserInGroup, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.GetGroup(db, data.ByID(id))
}

func ListIdentityGroups(c *gin.Context, userID uid.ID) ([]models.Group, error) {
	db, err := hasAuthorization(c, userID, isIdentitySelf, models.InfraAdminRole, models.InfraViewRole)
	if err != nil {
		return nil, err
	}

	return data.ListIdentityGroups(db, userID)
}
