package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get org for user")
	}

	var selectors = []data.SelectorFunc{
		orgSelector,
		data.ByPagination(pg),
	}

	if name != "" {
		selectors = append(selectors, data.ByName(name))
	}
	if userID != 0 {
		selectors = append(selectors, data.ByGroupMember(userID))
	}

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err == nil {
		return data.ListGroups(db, selectors...)
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
			return data.ListGroups(db, selectors...)
		}
	}

	return nil, err
}

func CreateGroup(c *gin.Context, group *models.Group) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "create", models.InfraAdminRole)
	}

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}
	group.OrganizationID = orgID

	return data.CreateGroup(db, group)
}

func GetGroup(c *gin.Context, id uid.ID) (*models.Group, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := hasAuthorization(c, id, isUserInGroup, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "group", "get", roles...)
	}

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get org for user")
	}

	return data.GetGroup(db, orgSelector, data.ByID(id))
}

func DeleteGroup(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "delete", models.InfraAdminRole)
	}

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return fmt.Errorf("Couldn't get org for user")
	}

	selectors := []data.SelectorFunc{
		orgSelector,
		data.ByID(id),
	}

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}

	return data.DeleteGroups(db, orgID, selectors...)
}

func checkIdentitiesInList(db *gorm.DB, orgSelector data.SelectorFunc, ids []uid.ID) ([]uid.ID, error) {
	identities, err := data.ListIdentities(db, orgSelector, data.ByIDs(ids))
	if err != nil {
		return nil, err
	}

	// return the original list if we found all of the IDs
	if len(identities) == len(ids) {
		return ids, nil
	}

	uidMap := make(map[uid.ID]bool)
	for _, ident := range identities {
		uidMap[ident.ID] = true
	}

	var uidStrList []string
	for _, id := range ids {
		_, ok := uidMap[id]
		if !ok {
			uidStrList = append(uidStrList, id.String())
		}
	}

	return nil, fmt.Errorf("%w: %s", internal.ErrBadRequest, "Couldn't find UIDs: "+strings.Join(uidStrList, ","))
}

func UpdateUsersInGroup(c *gin.Context, groupID uid.ID, uidsToAdd []uid.ID, uidsToRemove []uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return fmt.Errorf("Couldn't get org for user")
	}

	// Make sure the group exists before attempting to add/remove users
	_, err = data.GetGroup(db, orgSelector, data.ByID(groupID))
	if err != nil {
		return err
	}

	addIDList, err := checkIdentitiesInList(db, orgSelector, uidsToAdd)
	if err != nil {
		return err
	}

	rmIDList, err := checkIdentitiesInList(db, orgSelector, uidsToRemove)
	if err != nil {
		return err
	}

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}

	err = data.AddUsersToGroup(db, orgID, groupID, addIDList)
	if err != nil {
		return err
	}
	return data.RemoveUsersFromGroup(db, orgID, groupID, rmIDList)
}
