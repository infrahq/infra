package access

import (
	"errors"

	"github.com/gin-gonic/gin"

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

func ListGroups(c *gin.Context, name string, userID uid.ID, p *models.Pagination) ([]models.Group, error) {
	var selectors []data.SelectorFunc = []data.SelectorFunc{}
	if name != "" {
		selectors = append(selectors, data.ByName(name))
	}
	if userID != 0 {
		selectors = append(selectors, data.ByGroupMember(userID))
	}

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err == nil {
		return data.ListGroups(db, p, selectors...)
	}
	err = HandleAuthErr(err, "groups", "list", roles...)

	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own groups
		db := getDB(c)
		identity := AuthenticatedIdentity(c)
		switch {
		case identity == nil:
			return nil, err
		case userID == identity.ID:
			return data.ListGroups(db, p, selectors...)
		}
	}

	return nil, err
}

func CreateGroup(c *gin.Context, group *models.Group) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "create", models.InfraAdminRole)
	}

	return data.CreateGroup(db, group)
}

func GetGroup(c *gin.Context, id uid.ID) (*models.Group, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := hasAuthorization(c, id, isUserInGroup, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "group", "get", roles...)
	}

	return data.GetGroup(db, data.ByID(id))
}

func DeleteGroup(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "delete", models.InfraAdminRole)
	}

	selectors := []data.SelectorFunc{
		data.ByID(id),
	}

	return data.DeleteGroups(db, selectors...)
}
