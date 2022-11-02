package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListGroups(c *gin.Context, opts data.ListGroupOptions) ([]models.Group, error) {
	rCtx := GetRequestContext(c)

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	if err == nil {
		return data.ListGroups(rCtx.DBTxn, opts)
	}
	err = HandleAuthErr(err, "groups", "list", roles...)

	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own groups
		identity := rCtx.Authenticated.User
		switch {
		case identity == nil:
			return nil, err
		case opts.ByMemberID == identity.ID:
			return data.ListGroups(rCtx.DBTxn, opts)
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
	rCtx := GetRequestContext(c)
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "group", "get", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		if !userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, id) {
			return nil, err
		}
		// authorized by user belonging to the requested group
	} else if err != nil {
		return nil, err
	}
	return data.GetGroup(rCtx.DBTxn, data.GetGroupOptions{ByID: id})
}

func DeleteGroup(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "delete", models.InfraAdminRole)
	}
	return data.DeleteGroup(db, id)
}

func checkIdentitiesInList(db data.GormTxn, ids []uid.ID) ([]uid.ID, error) {
	if len(ids) == 0 {
		return ids, nil
	}

	identities, err := data.ListIdentities(db, data.ListIdentityOptions{ByIDs: ids})
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

	_, err = data.GetGroup(db, data.GetGroupOptions{ByID: groupID})
	if err != nil {
		return err
	}

	addIDList, err := checkIdentitiesInList(db, uidsToAdd)
	if err != nil {
		return err
	}

	rmIDList, err := checkIdentitiesInList(db, uidsToRemove)
	if err != nil {
		return err
	}

	if len(addIDList) > 0 {
		if err := data.AddUsersToGroup(db, groupID, addIDList); err != nil {
			return err
		}
	}

	if len(rmIDList) > 0 {
		if err := data.RemoveUsersFromGroup(db, groupID, rmIDList); err != nil {
			return err
		}
	}
	return nil
}
