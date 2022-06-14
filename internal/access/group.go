package access

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isUserInGroup is used by authorization checks to see if the calling user is requesting their own attributes
func isUserInGroup(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	user := AuthenticatedIdentity(c)

	if user != nil {
		return userInGroup(getDB(c), user.ID, requestedResourceID), nil
	}

	return false, nil
}

func ListGroups(c *gin.Context, name string, userID uid.ID, pg models.Pagination) ([]models.Group, error) {
	var selectors []data.SelectorFunc = []data.SelectorFunc{data.ByPagination(pg)}
	if name != "" {
		selectors = append(selectors, data.ByName(name))
	}
	if userID != 0 {
		selectors = append(selectors, data.ByGroupMember(userID))
	}

	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err == nil {
		return data.ListGroups(db, selectors...)
	}

	if errors.Is(err, internal.ErrForbidden) {
		// Allow an authenticated identity to view their own groups
		db := getDB(c)
		identity := AuthenticatedIdentity(c)
		switch {
		case identity == nil:
			return nil, err
		case userID == identity.ID:
			return data.ListGroups(db, selectors...)
		}
	}

	return nil, err
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
