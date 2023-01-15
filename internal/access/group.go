package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListGroups(rCtx RequestContext, name string, userID uid.ID, p *data.Pagination) ([]models.Group, error) {
	opts := data.ListGroupsOptions{
		ByName:        name,
		ByGroupMember: userID,
		Pagination:    p,
	}

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	err := IsAuthorized(rCtx, roles...)
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
		case userID == identity.ID:
			return data.ListGroups(rCtx.DBTxn, opts)
		}
	}

	return nil, err
}

func CreateGroup(rCtx RequestContext, group *models.Group) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "create", models.InfraAdminRole)
	}

	return data.CreateGroup(rCtx.DBTxn, group)
}

func GetGroup(rCtx RequestContext, opts data.GetGroupOptions) (*models.Group, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	err := IsAuthorized(rCtx, roles...)
	err = HandleAuthErr(err, "group", "get", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// get the group, but only to check if the user is in it
		group, err := data.GetGroup(rCtx.DBTxn, opts)
		if err != nil {
			return nil, err
		}
		if !userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, group.ID) {
			return nil, err
		}
		// authorized by user belonging to the requested group
	} else if err != nil {
		return nil, err
	}
	return data.GetGroup(rCtx.DBTxn, opts)
}

func DeleteGroup(rCtx RequestContext, id uid.ID) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "group", "delete", models.InfraAdminRole)
	}
	return data.DeleteGroup(rCtx.DBTxn, id)
}

func checkIdentitiesInList(db data.ReadTxn, ids []uid.ID) ([]uid.ID, error) {
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

func UpdateUsersInGroup(rCtx RequestContext, groupID uid.ID, uidsToAdd []uid.ID, uidsToRemove []uid.ID) error {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return err
	}

	_, err = data.GetGroup(rCtx.DBTxn, data.GetGroupOptions{ByID: groupID})
	if err != nil {
		return err
	}

	addIDList, err := checkIdentitiesInList(rCtx.DBTxn, uidsToAdd)
	if err != nil {
		return err
	}

	rmIDList, err := checkIdentitiesInList(rCtx.DBTxn, uidsToRemove)
	if err != nil {
		return err
	}

	if len(addIDList) > 0 {
		if err := data.AddUsersToGroup(rCtx.DBTxn, groupID, addIDList); err != nil {
			return err
		}
	}

	if len(rmIDList) > 0 {
		if err := data.RemoveUsersFromGroup(rCtx.DBTxn, groupID, rmIDList); err != nil {
			return err
		}
	}
	return nil
}
