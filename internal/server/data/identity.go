package data

import (
	"fmt"

	"github.com/ssoroka/slice"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func AssignIdentityToGroups(db *gorm.DB, user *models.Identity, provider *models.Provider, newGroups []string) error {
	pu, err := GetProviderUser(db, provider.ID, user.ID)
	if err != nil {
		return err
	}

	oldGroups := pu.Groups
	groupsToBeRemoved := slice.Subtract(oldGroups, newGroups)
	groupsToBeAdded := slice.Subtract(newGroups, oldGroups)

	pu.Groups = newGroups
	if err := save(db, pu); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// remove user from groups
	if len(groupsToBeRemoved) > 0 {
		err = db.Exec("delete from identities_groups where identity_id = ? and group_id in (select id from groups where name in (?))", user.ID, groupsToBeRemoved).Error
		if err != nil {
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

	var addIDs []struct {
		ID   uid.ID
		Name string
	}
	err = db.Table("groups").Select("id, name").Where("name in (?)", groupsToBeAdded).Scan(&addIDs).Error
	if err != nil {
		return fmt.Errorf("group ids: %w", err)
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
			group := &models.Group{Name: name}

			if err = CreateGroup(db, group); err != nil {
				return fmt.Errorf("create group: %w", err)
			}
			groupID = group.ID
		}

		var ids []uid.ID
		if err := db.Raw("SELECT identity_id FROM identities_groups WHERE identity_id = ? AND group_id = ?", user.ID, groupID).Scan(&ids).Error; err != nil {
			return fmt.Errorf("select: %w", handleError(err))
		}

		if len(ids) == 0 {
			// add user to group
			err = db.Exec("insert into identities_groups (identity_id, group_id) values (?, ?)", user.ID, groupID).Error
			if err != nil {
				return fmt.Errorf("insert: %w", handleError(err))
			}
		}

		user.Groups = append(user.Groups, models.Group{Model: models.Model{ID: groupID}, Name: name})
	}

	return nil
}

func CreateIdentity(db *gorm.DB, identity *models.Identity) error {
	return add(db, identity)
}

func GetIdentity(db *gorm.DB, selectors ...SelectorFunc) (*models.Identity, error) {
	return get[models.Identity](db, selectors...)
}

func ListIdentities(db *gorm.DB, selectors ...SelectorFunc) ([]models.Identity, error) {
	db = db.Order("name ASC")
	return list[models.Identity](db, selectors...)
}

func ListIdentitiesByGroup(db *gorm.DB, groupID uid.ID, selectors ...SelectorFunc) ([]models.Identity, error) {
	group, err := GetGroup(db.Preload("Identities", func(db *gorm.DB) *gorm.DB {
		for _, selector := range selectors {
			db = selector(db)
		}
		return db.Order("identities.name ASC")
	}), ByID(groupID))
	if err != nil {
		return nil, err
	}

	var identities []models.Identity
	identities = append(identities, group.Identities...)

	return identities, nil
}

func DeleteIdentity(db *gorm.DB, id uid.ID) error {
	return delete[models.Identity](db, id)
}

func DeleteIdentities(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListIdentities(db.Select("id"), selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, i := range toDelete {
		ids = append(ids, i.ID)

		err := DeleteGrants(db, BySubject(i.PolyID()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Identity](db, ByIDs(ids))
}

func SaveIdentity(db *gorm.DB, identity *models.Identity) error {
	return save(db, identity)
}
