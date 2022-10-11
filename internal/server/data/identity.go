package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/ssoroka/slice"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func AssignIdentityToGroups(tx GormTxn, user *models.Identity, provider *models.Provider, newGroups []string) error {
	pu, err := GetProviderUser(tx, provider.ID, user.ID)
	if err != nil {
		return err
	}

	oldGroups := pu.Groups
	groupsToBeRemoved := slice.Subtract(oldGroups, newGroups)
	groupsToBeAdded := slice.Subtract(newGroups, oldGroups)

	pu.Groups = newGroups
	pu.LastUpdate = time.Now().UTC()
	if err := UpdateProviderUser(tx, pu); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// remove user from groups
	if len(groupsToBeRemoved) > 0 {
		stmt := `DELETE FROM identities_groups WHERE identity_id = ? AND group_id in (
		   SELECT id FROM groups WHERE organization_id = ? AND name IN (?))`
		if _, err := tx.Exec(stmt, user.ID, tx.OrganizationID(), groupsToBeRemoved); err != nil {
			return err
		}
		for _, name := range groupsToBeRemoved {
			for i, g := range user.Groups {
				if g.Name == name {
					// remove from list
					user.Groups = append(user.Groups[:i], user.Groups[i+1:]...)
				}
			}
		}
	}

	type idNamePair struct {
		ID   uid.ID
		Name string
	}
	var addIDs []idNamePair

	stmt := `SELECT id, name FROM groups WHERE deleted_at is null AND name IN (?) AND organization_id = ?`
	rows, err := tx.Query(stmt, groupsToBeAdded, tx.OrganizationID())
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item idNamePair
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return err
		}
		addIDs = append(addIDs, item)
	}
	if rows.Err() != nil {
		return err
	}

	for _, name := range groupsToBeAdded {
		// find or create group
		var groupID uid.ID
		found := false
		for _, obj := range addIDs {
			if obj.Name == name {
				found = true
				groupID = obj.ID
				break
			}
		}
		if !found {
			group := &models.Group{
				Name:              name,
				CreatedByProvider: provider.ID,
			}

			if err = CreateGroup(tx, group); err != nil {
				return fmt.Errorf("create group: %w", err)
			}
			groupID = group.ID
		}

		var ids []uid.ID
		rows, err := tx.Query("SELECT identity_id FROM identities_groups WHERE identity_id = ? AND group_id = ?", user.ID, groupID)
		if err != nil {
			return err
		}

		for rows.Next() {
			var item uid.ID
			if err := rows.Scan(&item); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, item)
		}
		if rows.Err() != nil {
			return err
		}

		if len(ids) == 0 {
			// add user to group
			_, err = tx.Exec("INSERT INTO identities_groups (identity_id, group_id) VALUES (?, ?)", user.ID, groupID)
			if err != nil {
				return fmt.Errorf("insert: %w", handleError(err))
			}
		}

		user.Groups = append(user.Groups, models.Group{Model: models.Model{ID: groupID}, Name: name})
	}

	return nil
}

func CreateIdentity(db GormTxn, identity *models.Identity) error {
	return add(db, identity)
}

func GetIdentity(db GormTxn, selectors ...SelectorFunc) (*models.Identity, error) {
	return get[models.Identity](db, selectors...)
}

func SetIdentityVerified(db GormTxn, token string) error {
	q := querybuilder.New(`UPDATE identities SET verified = true`)
	q.B("WHERE verified = ? AND verification_token = ? AND organization_id = ?", false, token, db.OrganizationID())

	_, err := db.Exec(q.String(), q.Args...)
	if err != nil {
		return err
	}

	return nil
}

func ListIdentities(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Identity, error) {
	return list[models.Identity](db, p, selectors...)
}

func SaveIdentity(db GormTxn, identity *models.Identity) error {
	return save(db, identity)
}

func DeleteIdentities(tx GormTxn, providerID uid.ID, selectors ...SelectorFunc) error {
	selectID := func(db *gorm.DB) *gorm.DB {
		return db.Select("id")
	}
	selectors = append([]SelectorFunc{selectID}, selectors...)
	toDelete, err := ListIdentities(tx, nil, selectors...)
	if err != nil {
		return err
	}

	ids, err := deleteReferencesToIdentities(tx, providerID, toDelete)
	if err != nil {
		return fmt.Errorf("remove identities: %w", err)
	}

	if len(ids) > 0 {
		return deleteAll[models.Identity](tx, ByIDs(ids))
	}

	return nil
}

func deleteReferencesToIdentities(tx GormTxn, providerID uid.ID, toDelete []models.Identity) (unreferencedIdentityIDs []uid.ID, err error) {
	for _, i := range toDelete {
		if err := DeleteAccessKeys(tx, DeleteAccessKeysOptions{ByIssuedForID: i.ID, ByProviderID: providerID}); err != nil {
			return []uid.ID{}, fmt.Errorf("delete identity access keys: %w", err)
		}
		if providerID == InfraProvider(tx).ID {
			// if an identity does not have credentials in the Infra provider this won't be found, but we can proceed
			credential, err := GetCredential(tx, ByIdentityID(i.ID))
			if err != nil && !errors.Is(err, internal.ErrNotFound) {
				return []uid.ID{}, fmt.Errorf("get delete identity creds: %w", err)
			}
			if credential != nil {
				err := DeleteCredential(tx, credential.ID)
				if err != nil {
					return []uid.ID{}, fmt.Errorf("delete identity creds: %w", err)
				}
			}
		}
		if err := DeleteProviderUsers(tx, DeleteProviderUsersOptions{ByIdentityID: i.ID, ByProviderID: providerID}); err != nil {
			return []uid.ID{}, fmt.Errorf("remove provider user: %w", err)
		}

		// if this identity no longer exists in any identity providers then remove all their references
		user, err := GetIdentity(tx, Preload("Providers"), ByID(i.ID))
		if err != nil {
			return []uid.ID{}, fmt.Errorf("check user providers: %w", err)
		}

		if len(user.Providers) == 0 {
			groups, err := ListGroups(tx, nil, ByGroupMember(i.ID))
			if err != nil {
				return []uid.ID{}, fmt.Errorf("list groups for identity: %w", err)
			}
			for _, group := range groups {
				err = RemoveUsersFromGroup(tx, group.ID, []uid.ID{i.ID})
				if err != nil {
					return []uid.ID{}, fmt.Errorf("delete group membership for identity: %w", err)
				}
			}
			err = DeleteGrants(tx, DeleteGrantsOptions{BySubject: uid.NewIdentityPolymorphicID(i.ID)})
			if err != nil {
				return []uid.ID{}, fmt.Errorf("delete identity creds: %w", err)
			}
			unreferencedIdentityIDs = append(unreferencedIdentityIDs, user.ID)
		}
	}
	return unreferencedIdentityIDs, nil
}
