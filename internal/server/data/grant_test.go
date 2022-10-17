package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestCreateGrant(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			actual := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "infra",
				CreatedBy: uid.ID(1091),
			}
			err := CreateGrant(tx, &actual)
			assert.NilError(t, err)
			assert.Assert(t, actual.ID != 0)

			expected := models.Grant{
				Model: models.Model{
					ID:        uid.ID(999),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
				Subject:            "i:1234567",
				Privilege:          "view",
				Resource:           "infra",
				CreatedBy:          uid.ID(1091),
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("duplicate grant", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			g := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "infra",
			}
			err := CreateGrant(tx, &g)
			assert.NilError(t, err)

			g2 := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "infra",
			}
			err = CreateGrant(tx, &g2)
			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			assert.DeepEqual(t, ucErr, UniqueConstraintError{Table: "grants"})

			grants, err := ListGrants(tx, ListGrantsOptions{
				BySubject:  "i:1234567",
				ByResource: "infra",
			})
			assert.NilError(t, err)
			assert.Assert(t, is.Len(grants, 1))

			g3 := models.Grant{
				Subject:   "i:1234567",
				Privilege: "edit",
				Resource:  "infra",
			}
			// check that unique constraint needs all three fields
			err = CreateGrant(tx, &g3)
			assert.NilError(t, err)
		})
		t.Run("notify", func(t *testing.T) {
			ctx := context.Background()
			listener, err := ListenForGrantsNotify(ctx, db, ListenForGrantsOptions{
				ByDestination: "match",
				OrgID:         defaultOrganizationID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			g := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "match",
			}
			err = CreateGrant(tx, &g)
			assert.NilError(t, err)
			assert.NilError(t, tx.Commit())

			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			err = listener.WaitForNotification(ctx)
			assert.NilError(t, err)
		})
	})
}

func TestDeleteGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		var startUpdateIndex int64 = 10001

		t.Run("empty options", func(t *testing.T) {
			err := DeleteGrants(db, DeleteGrantsOptions{})
			assert.ErrorContains(t, err, "requires an ID to delete")
		})
		t.Run("by id", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			grant := &models.Grant{Subject: "i:any", Privilege: "view", Resource: "any"}
			toKeep := &models.Grant{Subject: "i:any2", Privilege: "view", Resource: "any"}
			createGrants(t, tx, grant, toKeep)

			err := DeleteGrants(tx, DeleteGrantsOptions{ByID: grant.ID})
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "any"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			maxIndex, err := GrantsMaxUpdateIndex(tx, GrantsMaxUpdateIndexOptions{ByDestination: "any"})
			assert.NilError(t, err)
			assert.Equal(t, maxIndex, startUpdateIndex+3) // 2 inserts, 1 delete
			startUpdateIndex = maxIndex
		})
		t.Run("by subject", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			grant1 := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "any"}
			grant2 := &models.Grant{Subject: "i:any1", Privilege: "edit", Resource: "any"}
			toKeep := &models.Grant{Subject: "i:any2", Privilege: "view", Resource: "any"}
			createGrants(t, tx, grant1, grant2, toKeep)

			otherOrgGrant := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "any"}
			createGrants(t, tx.WithOrgID(otherOrg.ID), otherOrgGrant)

			err := DeleteGrants(tx, DeleteGrantsOptions{BySubject: grant1.Subject})
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "any"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			maxIndex, err := GrantsMaxUpdateIndex(tx, GrantsMaxUpdateIndexOptions{ByDestination: "any"})
			assert.NilError(t, err)
			assert.Equal(t, maxIndex, startUpdateIndex+6) // 4 inserts, 2 deletes
			startUpdateIndex = maxIndex

			// other org still has the grant
			actual, err = ListGrants(tx.WithOrgID(otherOrg.ID), ListGrantsOptions{ByDestination: "any"})
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 1)
		})
		t.Run("by created_by and not ids", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			createdBy := uid.ID(9234)
			grant1 := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "any", CreatedBy: createdBy}
			grant2 := &models.Grant{Subject: "i:any2", Privilege: "view", Resource: "any", CreatedBy: createdBy}
			toKeep1 := &models.Grant{Subject: "i:any3", Privilege: "view", Resource: "any", CreatedBy: createdBy}
			toKeep2 := &models.Grant{Subject: "i:any4", Privilege: "view", Resource: "any"}
			createGrants(t, tx, grant1, grant2, toKeep1, toKeep2)

			err := DeleteGrants(tx, DeleteGrantsOptions{
				ByCreatedBy: createdBy,
				NotIDs:      []uid.ID{toKeep1.ID},
			})
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "any"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep1.ID}},
				{Model: models.Model{ID: toKeep2.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			maxIndex, err := GrantsMaxUpdateIndex(tx, GrantsMaxUpdateIndexOptions{ByDestination: "any"})
			assert.NilError(t, err)
			assert.Equal(t, maxIndex, startUpdateIndex+6) // 4 inserts, 2 deletes
			startUpdateIndex = maxIndex
		})
		t.Run("notify", func(t *testing.T) {
			g := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "match.a.resource",
			}
			assert.NilError(t, CreateGrant(db, &g))

			ctx := context.Background()
			listener, err := ListenForGrantsNotify(ctx, db, ListenForGrantsOptions{
				ByDestination: "match",
				OrgID:         defaultOrganizationID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			err = DeleteGrants(tx, DeleteGrantsOptions{BySubject: "i:1234567"})
			assert.NilError(t, err)
			assert.NilError(t, tx.Commit())

			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			t.Cleanup(cancel)

			err = listener.WaitForNotification(ctx)
			assert.NilError(t, err)
		})
	})
}

func createGrants(t *testing.T, tx WriteTxn, grants ...*models.Grant) {
	t.Helper()
	for _, grant := range grants {
		assert.NilError(t, CreateGrant(tx, grant))
	}
}

func TestGetGrant(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		grant1 := &models.Grant{
			Subject:   "i:any1",
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant2 := &models.Grant{
			Subject:   "i:any2",
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		createGrants(t, tx, grant1, grant2)

		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(tx, otherOrg))
		other := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "any"}
		createGrants(t, tx.WithOrgID(otherOrg.ID), other)

		t.Run("default options", func(t *testing.T) {
			_, err := GetGrant(tx, GetGrantOptions{})
			assert.ErrorContains(t, err, "GetGrant requires an ID")
		})
		t.Run("not found", func(t *testing.T) {
			_, err := GetGrant(tx, GetGrantOptions{ByID: uid.ID(404)})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("not found soft deleted", func(t *testing.T) {
			deleted := &models.Grant{
				Model:              models.Model{ID: 1234},
				OrganizationMember: models.OrganizationMember{},
				Subject:            "i:someone",
				Privilege:          "view",
				Resource:           "any",
			}
			deleted.DeletedAt.Time = time.Now()
			deleted.DeletedAt.Valid = true
			createGrants(t, tx, deleted)

			_, err := GetGrant(tx, GetGrantOptions{ByID: uid.ID(1234)})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("wrong org", func(t *testing.T) {
			_, err := GetGrant(tx, GetGrantOptions{ByID: uid.ID(other.ID)})
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("by id", func(t *testing.T) {
			actual, err := GetGrant(tx, GetGrantOptions{ByID: grant1.ID})
			assert.NilError(t, err)

			expected := &models.Grant{
				Model: models.Model{
					ID:        grant1.ID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: db.DefaultOrg.ID},
				Subject:            "i:any1",
				Privilege:          "view",
				Resource:           "any",
				CreatedBy:          uid.ID(777),
				UpdateIndex:        10001,
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("by subject, privilege, resource", func(t *testing.T) {
			actual, err := GetGrant(tx, GetGrantOptions{
				BySubject:   "i:any1",
				ByPrivilege: "view",
				ByResource:  "any",
			})
			assert.NilError(t, err)

			expected := &models.Grant{
				Model: models.Model{
					ID:        grant1.ID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{OrganizationID: db.DefaultOrg.ID},
				Subject:            "i:any1",
				Privilege:          "view",
				Resource:           "any",
				CreatedBy:          uid.ID(777),
				UpdateIndex:        10001,
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
	})
}

func TestListGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		user := &models.Identity{Name: "usera@example.com"}
		createIdentities(t, tx, user)

		grant1 := &models.Grant{
			Subject:   "i:userchar",
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant2 := &models.Grant{
			Subject:   "i:userbeta",
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant3 := &models.Grant{
			Subject:   "i:userchar",
			Privilege: "admin",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant4 := &models.Grant{
			Subject:   "i:userchar",
			Privilege: "view",
			Resource:  "infra",
			CreatedBy: uid.ID(777),
		}
		grant5 := &models.Grant{
			Subject:   "i:userdelta",
			Privilege: "logs",
			Resource:  "any.namespace",
			CreatedBy: uid.ID(777),
		}
		deleted := &models.Grant{
			Subject:   "i:userchar",
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true
		createGrants(t, tx, grant1, grant2, grant3, grant4, grant5, deleted)

		userID, err := uid.Parse([]byte("userchar"))
		assert.NilError(t, err)

		assert.NilError(t, AddUsersToGroup(tx, uid.ID(111), []uid.ID{userID}))
		assert.NilError(t, AddUsersToGroup(tx, uid.ID(112), []uid.ID{userID}))
		assert.NilError(t, AddUsersToGroup(tx, uid.ID(113), []uid.ID{uid.ID(777)}))

		gGrant1 := &models.Grant{
			Subject:   uid.NewGroupPolymorphicID(111),
			Privilege: "view",
			Resource:  "anyother",
			CreatedBy: uid.ID(777),
		}
		gGrant2 := &models.Grant{
			Subject:   uid.NewGroupPolymorphicID(112),
			Privilege: "admin",
			Resource:  "shared",
			CreatedBy: uid.ID(777),
		}
		gGrant3 := &models.Grant{
			Subject:   uid.NewGroupPolymorphicID(113),
			Privilege: "admin",
			Resource:  "special",
			CreatedBy: uid.ID(777),
		}
		createGrants(t, tx, gGrant1, gGrant2, gGrant3)

		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(tx, otherOrg))

		otherOrgGrant := &models.Grant{
			Subject:            "i:userchar",
			Privilege:          "view",
			Resource:           "any",
			CreatedBy:          uid.ID(778),
			OrganizationMember: models.OrganizationMember{OrganizationID: otherOrg.ID},
		}
		createGrants(t, tx.WithOrgID(otherOrg.ID), otherOrgGrant)

		connectorUser := InfraConnectorIdentity(db)

		connector, err := GetGrant(tx, GetGrantOptions{
			BySubject:   uid.NewIdentityPolymorphicID(connectorUser.ID),
			ByPrivilege: models.InfraConnectorRole,
			ByResource:  "infra",
		})
		assert.NilError(t, err)

		t.Run("default", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{})
			assert.NilError(t, err)

			expected := []models.Grant{
				*connector,
				*grant1,
				*grant2,
				*grant3,
				*grant4,
				*grant5,
				*gGrant1,
				*gGrant2,
				*gGrant3,
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by subject", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{BySubject: "i:userchar"})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant3, *grant4}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by resource", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{ByResource: "any"})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant2, *grant3}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by resource and privilege", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{
				ByResource:   "any",
				ByPrivileges: []string{"view"},
			})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant2}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("with multiple privileges", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{
				ByResource:   "any",
				ByPrivileges: []string{"view", "admin"},
			})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant2, *grant3}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("by subject with include inherited", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{
				BySubject:                  "i:userchar",
				IncludeInheritedFromGroups: true,
			})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant3, *grant4, *gGrant1, *gGrant2}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("exclude connector grant", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{ExcludeConnectorGrant: true})
			assert.NilError(t, err)

			expected := []models.Grant{
				*grant1,
				*grant2,
				*grant3,
				*grant4,
				*grant5,
				*gGrant1,
				*gGrant2,
				*gGrant3,
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("with pagination", func(t *testing.T) {
			pagination := &Pagination{Page: 2, Limit: 3}
			actual, err := ListGrants(tx, ListGrantsOptions{Pagination: pagination})
			assert.NilError(t, err)

			expected := []models.Grant{*grant3, *grant4, *grant5}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			expectedPagination := &Pagination{Page: 2, Limit: 3, TotalCount: 9}
			assert.DeepEqual(t, pagination, expectedPagination)
		})
		t.Run("by resource with pagination", func(t *testing.T) {
			pagination := &Pagination{Page: 1, Limit: 2}
			actual, err := ListGrants(tx, ListGrantsOptions{
				Pagination: pagination,
				ByResource: "any",
			})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant2}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			expectedPagination := &Pagination{Page: 1, Limit: 2, TotalCount: 3}
			assert.DeepEqual(t, pagination, expectedPagination)
		})
		t.Run("by destination", func(t *testing.T) {
			actual, err := ListGrants(tx, ListGrantsOptions{
				ByDestination: "any",
			})
			assert.NilError(t, err)

			expected := []models.Grant{*grant1, *grant2, *grant3, *grant5}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
	})
}

func TestGrantsMaxUpdateIndex(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("no results match the query", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			idx, err := GrantsMaxUpdateIndex(tx, GrantsMaxUpdateIndexOptions{ByDestination: "nope"})
			assert.NilError(t, err)
			assert.Equal(t, idx, int64(1))
		})
	})
}
