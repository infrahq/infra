package data

import (
	"errors"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestCreateGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			actual := models.Group{
				Name:              "Everyone",
				CreatedBy:         uid.ID(1011),
				CreatedByProvider: uid.ID(2022),
			}
			err := CreateGroup(tx, &actual)
			assert.NilError(t, err)
			expected := models.Group{
				Model: models.Model{
					ID:        uid.ID(999),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{
					OrganizationID: defaultOrganizationID,
				},
				Name:              "Everyone",
				CreatedBy:         uid.ID(1011),
				CreatedByProvider: uid.ID(2022),
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("duplicate name", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			err := CreateGroup(tx, &models.Group{Name: "Everyone"})
			assert.NilError(t, err)

			err = CreateGroup(tx, &models.Group{Name: "Everyone"})
			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			expectedErr := UniqueConstraintError{Table: "groups", Column: "name"}
			assert.DeepEqual(t, ucErr, expectedErr)
		})
	})
}

func createGroups(t *testing.T, db GormTxn, groups ...*models.Group) {
	t.Helper()
	for i := range groups {
		err := CreateGroup(db, groups[i])
		assert.NilError(t, err, groups[i].Name)
	}
}

func TestGetGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		group, err := GetGroup(db, ByName(everyone.Name))
		assert.NilError(t, err)
		assert.Assert(t, 0 != group.ID)
	})
}

func TestListGroups(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		firstUser := models.Identity{
			Name:   "firstly",
			Groups: []models.Group{everyone, engineers},
		}
		secondUser := models.Identity{
			Name:   "secondarly",
			Groups: []models.Group{everyone, product},
		}
		createIdentities(t, db, &firstUser, &secondUser)

		t.Run("all", func(t *testing.T) {
			actual, err := ListGroups(db, nil)
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
				{Name: "Everyone"},
				{Name: "Product"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by name", func(t *testing.T) {
			actual, err := ListGroups(db, nil, ByName(engineers.Name))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by identity membership", func(t *testing.T) {
			actual, err := ListGroups(db, nil, ByGroupMember(firstUser.ID))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
				{Name: "Everyone"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})
	})
}

var cmpGroupShallow = gocmp.Comparer(func(x, y models.Group) bool {
	return x.Name == y.Name
})

func TestDeleteGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(tx, otherOrg))

		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)
		createGroups(t, tx, &everyone, &engineers, &product)

		someone := &models.Identity{
			Name:   "someone@example.com",
			Groups: []models.Group{everyone},
		}
		createIdentities(t, tx, someone)

		groupGrant := &models.Grant{
			Subject:   uid.NewGroupPolymorphicID(everyone.ID),
			Privilege: "admin",
			Resource:  "any",
		}
		createGrants(t, tx, groupGrant)

		otherOrgGroup := &models.Group{Name: "Everyone"}
		createGroups(t, tx.WithOrgID(otherOrg.ID), otherOrgGroup)

		t.Run("success", func(t *testing.T) {
			_, err := GetGroup(tx, ByID(everyone.ID))
			assert.NilError(t, err)

			err = DeleteGroup(tx, everyone.ID)
			assert.NilError(t, err)

			_, err = GetGroup(tx, ByID(everyone.ID))
			assert.Error(t, err, "record not found")

			// deleting a group should not delete unrelated groups
			_, err = GetGroup(tx, ByID(engineers.ID))
			assert.NilError(t, err)
			_, err = GetGroup(tx.WithOrgID(otherOrg.ID), ByID(otherOrgGroup.ID))
			assert.NilError(t, err)

			// grants and group membership should also be removed.
			users, err := ListIdentities(tx, ListIdentityOptions{ByGroupID: everyone.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, users, []models.Identity{})

			grants, err := ListGrants(tx, ListGrantsOptions{BySubject: groupGrant.Subject})
			assert.NilError(t, err)
			assert.DeepEqual(t, grants, []models.Grant{}, cmpopts.EquateEmpty())
		})
		t.Run("delete non-existent", func(t *testing.T) {
			err := DeleteGroup(tx, uid.ID(1234))
			assert.NilError(t, err)
		})
		t.Run("delete already soft-deleted", func(t *testing.T) {
			err := DeleteGroup(tx, everyone.ID)
			assert.NilError(t, err)
			err = DeleteGroup(tx, everyone.ID)
			assert.NilError(t, err)
		})
	})
}

func TestRecreateGroupSameName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		err := DeleteGroup(db, everyone.ID)
		assert.NilError(t, err)

		err = CreateGroup(db, &models.Group{Name: everyone.Name})
		assert.NilError(t, err)
	})
}

func TestAddUsersToGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		everyone := models.Group{Name: "Everyone"}
		other := models.Group{Name: "Other"}
		createGroups(t, db, &everyone, &other)

		var (
			bond = models.Identity{
				Name:   "jbond@infrahq.com",
				Groups: []models.Group{everyone},
			}
			bourne = models.Identity{Name: "jbourne@infrahq.com"}
			bauer  = models.Identity{Name: "jbauer@infrahq.com"}
			forth  = models.Identity{
				Name:   "forth@example.com",
				Groups: []models.Group{everyone},
			}
		)

		createIdentities(t, db, &bond, &bourne, &bauer, &forth)

		t.Run("add identities to group", func(t *testing.T) {
			actual, err := ListIdentities(db, ListIdentityOptions{ByGroupID: everyone.ID})
			assert.NilError(t, err)
			expected := []models.Identity{forth, bond}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)

			err = AddUsersToGroup(db, everyone.ID, []uid.ID{bourne.ID, bauer.ID, forth.ID})
			assert.NilError(t, err)

			actual, err = ListIdentities(db, ListIdentityOptions{ByGroupID: everyone.ID})
			assert.NilError(t, err)
			expected = []models.Identity{forth, bauer, bond, bourne}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)

			actual, err = ListIdentities(db, ListIdentityOptions{ByGroupID: other.ID})
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 0)
		})
	})
}

func TestRemoveUsersFromGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		everyone := models.Group{Name: "Everyone"}
		other := models.Group{Name: "Other"}
		createGroups(t, tx, &everyone, &other)

		bond := models.Identity{
			Name:   "jbond@infrahq.com",
			Groups: []models.Group{everyone, other},
		}
		bourne := models.Identity{
			Name:   "jbourne@infrahq.com",
			Groups: []models.Group{everyone, other},
		}
		bauer := models.Identity{
			Name:   "jbauer@infrahq.com",
			Groups: []models.Group{everyone, other},
		}
		forth := models.Identity{
			Name:   "forth@example.com",
			Groups: []models.Group{everyone},
		}
		createIdentities(t, tx, &bond, &bourne, &bauer, &forth)

		users, err := ListIdentities(tx, ListIdentityOptions{ByGroupID: everyone.ID})
		assert.NilError(t, err)
		assert.Equal(t, len(users), 4)

		users, err = ListIdentities(tx, ListIdentityOptions{ByGroupID: other.ID})
		assert.NilError(t, err)
		assert.Equal(t, len(users), 3)

		err = RemoveUsersFromGroup(tx, everyone.ID, []uid.ID{bond.ID, bourne.ID, forth.ID})
		assert.NilError(t, err)

		actual, err := ListIdentities(tx, ListIdentityOptions{ByGroupID: everyone.ID})
		assert.NilError(t, err)
		expected := []models.Identity{bauer}
		assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)

		actual, err = ListIdentities(tx, ListIdentityOptions{ByGroupID: other.ID})
		assert.NilError(t, err)
		expected = []models.Identity{bauer, bond, bourne}
		assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
	})
}
