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
				Subject:   models.NewSubjectForUser(1234567),
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
				Subject:            models.NewSubjectForUser(1234567),
				Privilege:          "view",
				Resource:           "infra",
				CreatedBy:          uid.ID(1091),
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("duplicate grant", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			g := models.Grant{
				Subject:   models.NewSubjectForUser(1234567),
				Privilege: "view",
				Resource:  "infra",
			}
			err := CreateGrant(tx, &g)
			assert.NilError(t, err)

			g2 := models.Grant{
				Subject:   models.NewSubjectForUser(1234567),
				Privilege: "view",
				Resource:  "infra",
			}
			err = CreateGrant(tx, &g2)
			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			assert.DeepEqual(t, ucErr, UniqueConstraintError{Table: "grants"})

			grants, err := ListGrants(tx, ListGrantsOptions{
				BySubject:  models.NewSubjectForUser(1234567),
				ByResource: "infra",
			})
			assert.NilError(t, err)
			assert.Assert(t, is.Len(grants, 1))

			g3 := models.Grant{
				Subject:   models.NewSubjectForUser(1234567),
				Privilege: "edit",
				Resource:  "infra",
			}
			// check that unique constraint needs all three fields
			err = CreateGrant(tx, &g3)
			assert.NilError(t, err)
		})
		t.Run("notify", func(t *testing.T) {
			dest := &models.Destination{Name: "match", Kind: "other"}
			createDestinations(t, db, dest)

			ctx := context.Background()
			listener, err := ListenForNotify(ctx, db, ListenChannelGrantsByDestination{
				DestinationID: dest.ID,
				OrgID:         defaultOrganizationID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			g := models.Grant{
				Subject:   models.NewSubjectForUser(1234567),
				Privilege: "view",
				Resource:  "match",
			}
			err = CreateGrant(tx, &g)
			assert.NilError(t, err)
			assert.NilError(t, tx.Commit())

			ctx, cancel := context.WithTimeout(ctx, time.Second)
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

			grant := &models.Grant{Subject: models.NewSubjectForUser(4444), Privilege: "view", Resource: "any"}
			toKeep := &models.Grant{Subject: models.NewSubjectForUser(6666), Privilege: "view", Resource: "any"}
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

			grant1 := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any"}
			grant2 := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "edit", Resource: "any"}
			toKeep := &models.Grant{Subject: models.NewSubjectForUser(6666), Privilege: "view", Resource: "any"}
			createGrants(t, tx, grant1, grant2, toKeep)

			otherOrgGrant := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any"}
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
		t.Run("by destination", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			toKeep := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "somethingelse"}
			createGrants(t, tx, toKeep)

			grants := []*models.Grant{
				{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any"},
				{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any.one"},
				{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any.two"},
			}
			createGrants(t, tx, grants...)

			actual, err := ListGrants(tx, ListGrantsOptions{BySubject: models.NewSubjectForUser(5555)})
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 4)

			err = DeleteGrants(tx, DeleteGrantsOptions{ByDestination: "any"})
			assert.NilError(t, err)

			actual, err = ListGrants(tx, ListGrantsOptions{BySubject: models.NewSubjectForUser(5555)})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
		t.Run("notify", func(t *testing.T) {
			g := models.Grant{
				Subject:   models.NewSubjectForUser(1234567),
				Privilege: "view",
				Resource:  "match.a.resource",
			}
			assert.NilError(t, CreateGrant(db, &g))

			dest := &models.Destination{Name: "match", Kind: "other"}
			createDestinations(t, db, dest)

			ctx := context.Background()
			listener, err := ListenForNotify(ctx, db, ListenChannelGrantsByDestination{
				DestinationID: dest.ID,
				OrgID:         defaultOrganizationID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			err = DeleteGrants(tx, DeleteGrantsOptions{BySubject: models.NewSubjectForUser(1234567)})
			assert.NilError(t, err)
			assert.NilError(t, tx.Commit())

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			t.Cleanup(cancel)

			err = listener.WaitForNotification(ctx)
			assert.NilError(t, err)
		})
	})
}

func TestUpdateGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		dest1 := &models.Destination{Name: "foo", Kind: "ssh"}
		dest2 := &models.Destination{Name: "foo2", Kind: "ssh"}
		createDestinations(t, db, dest1, dest2)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		var startUpdateIndex int64 = 10001

		t.Run("success add", func(t *testing.T) {
			listener, err := ListenForNotify(ctx, db, ListenChannelGrantsByDestination{
				OrgID:         db.DefaultOrg.ID,
				DestinationID: dest1.ID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			addGrants := []*models.Grant{
				{Subject: models.NewSubjectForUser(7654321), Privilege: "view", Resource: "foo", CreatedBy: uid.ID(1091)},
				{Subject: models.NewSubjectForUser(1234567), Privilege: "admin", Resource: "foo", CreatedBy: uid.ID(1091)},
			}
			rmGrants := []*models.Grant{}

			err = UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)
			assert.Assert(t, addGrants[0].ID != 0)

			expected := []models.Grant{
				{
					Model: models.Model{
						ID:        uid.ID(999),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
					Subject:            models.NewSubjectForUser(7654321),
					Privilege:          "view",
					Resource:           "foo",
					CreatedBy:          uid.ID(1091),
					UpdateIndex:        startUpdateIndex + 1,
				},
				{
					Model: models.Model{
						ID:        uid.ID(999),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
					Subject:            models.NewSubjectForUser(1234567),
					Privilege:          "admin",
					Resource:           "foo",
					CreatedBy:          uid.ID(1091),
					UpdateIndex:        startUpdateIndex + 2,
				},
			}

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, expected, cmpModel)

			// check for idempotency
			err = UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)
			actual, err = ListGrants(tx, ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, expected, cmpModel)

			assert.NilError(t, tx.Commit())

			ctx, cancel = context.WithTimeout(ctx, time.Second)
			err = listener.WaitForNotification(ctx)
			assert.NilError(t, err)
		})
		t.Run("success delete", func(t *testing.T) {
			listener, err := ListenForNotify(ctx, db, ListenChannelGrantsByDestination{
				OrgID:         db.DefaultOrg.ID,
				DestinationID: dest2.ID,
			})
			assert.NilError(t, err)
			t.Cleanup(func() {
				assert.NilError(t, listener.Release(context.Background()))
			})

			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			grant1 := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "foo2"}
			grant2 := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "edit", Resource: "foo2"}
			toKeep := &models.Grant{Subject: models.NewSubjectForUser(6666), Privilege: "view", Resource: "foo2"}
			createGrants(t, tx, grant1, grant2, toKeep)

			otherOrgGrant := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "foo2"}
			createGrants(t, tx.WithOrgID(otherOrg.ID), otherOrgGrant)

			addGrants := []*models.Grant{}
			rmGrants := []*models.Grant{grant1, grant2}

			err = UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "foo2"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			// check for idempotency
			err = UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err = ListGrants(tx, ListGrantsOptions{ByDestination: "foo2"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			// other org still has the grant
			actual, err = ListGrants(tx.WithOrgID(otherOrg.ID), ListGrantsOptions{ByDestination: "foo2"})
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 1)

			assert.NilError(t, tx.Commit())

			ctx, cancel = context.WithTimeout(ctx, time.Second)
			err = listener.WaitForNotification(ctx)
			assert.NilError(t, err)
		})
		t.Run("success create and delete", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			addGrants := []*models.Grant{
				{Subject: models.NewSubjectForUser(7654321), Privilege: "view", Resource: "foo3", CreatedBy: uid.ID(1091)},
				{Subject: models.NewSubjectForUser(1234567), Privilege: "admin", Resource: "foo3", CreatedBy: uid.ID(1091)},
			}
			rmGrants := []*models.Grant{
				{Subject: models.NewSubjectForUser(7654321), Privilege: "view", Resource: "foo3", CreatedBy: uid.ID(1091)},
				{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "foo3"},
				{Subject: models.NewSubjectForUser(5555), Privilege: "edit", Resource: "foo3"},
			}

			err := UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "foo3"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{
					Model: models.Model{
						ID:        uid.ID(999),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
					Subject:            models.NewSubjectForUser(1234567),
					Privilege:          "admin",
					Resource:           "foo3",
					CreatedBy:          uid.ID(1091),
					UpdateIndex:        startUpdateIndex + 12,
				},
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
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
			Subject:   models.NewSubjectForUser(5555),
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant2 := &models.Grant{
			Subject:   models.NewSubjectForUser(6666),
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		createGrants(t, tx, grant1, grant2)

		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(tx, otherOrg))
		other := &models.Grant{Subject: models.NewSubjectForUser(5555), Privilege: "view", Resource: "any"}
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
				Subject:            models.NewSubjectForUser(72422),
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
				Subject:            models.NewSubjectForUser(5555),
				Privilege:          "view",
				Resource:           "any",
				CreatedBy:          uid.ID(777),
				UpdateIndex:        10001,
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("by subject, privilege, resource", func(t *testing.T) {
			actual, err := GetGrant(tx, GetGrantOptions{
				BySubject:   models.NewSubjectForUser(5555),
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
				Subject:            models.NewSubjectForUser(5555),
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

		userID := uid.ID(71234)

		grant1 := &models.Grant{
			Subject:   models.NewSubjectForUser(userID),
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant2 := &models.Grant{
			Subject:   models.NewSubjectForUser(8235),
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant3 := &models.Grant{
			Subject:   models.NewSubjectForUser(userID),
			Privilege: "admin",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		grant4 := &models.Grant{
			Subject:   models.NewSubjectForUser(userID),
			Privilege: "view",
			Resource:  "infra",
			CreatedBy: uid.ID(777),
		}
		grant5 := &models.Grant{
			Subject:   models.NewSubjectForUser(9366),
			Privilege: "logs",
			Resource:  "any.namespace",
			CreatedBy: uid.ID(777),
		}
		deleted := &models.Grant{
			Subject:   models.NewSubjectForUser(userID),
			Privilege: "view",
			Resource:  "any",
			CreatedBy: uid.ID(777),
		}
		deleted.DeletedAt.Time = time.Now()
		deleted.DeletedAt.Valid = true
		createGrants(t, tx, grant1, grant2, grant3, grant4, grant5, deleted)

		assert.NilError(t, AddUsersToGroup(tx, uid.ID(111), []uid.ID{userID}))
		assert.NilError(t, AddUsersToGroup(tx, uid.ID(112), []uid.ID{userID}))
		assert.NilError(t, AddUsersToGroup(tx, uid.ID(113), []uid.ID{uid.ID(777)}))

		gGrant1 := &models.Grant{
			Subject:   models.NewSubjectForGroup(111),
			Privilege: "view",
			Resource:  "anyother",
			CreatedBy: uid.ID(777),
		}
		gGrant2 := &models.Grant{
			Subject:   models.NewSubjectForGroup(112),
			Privilege: "admin",
			Resource:  "shared",
			CreatedBy: uid.ID(777),
		}
		gGrant3 := &models.Grant{
			Subject:   models.NewSubjectForGroup(113),
			Privilege: "admin",
			Resource:  "special",
			CreatedBy: uid.ID(777),
		}
		createGrants(t, tx, gGrant1, gGrant2, gGrant3)

		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(tx, otherOrg))

		otherOrgGrant := &models.Grant{
			Subject:            models.NewSubjectForUser(userID),
			Privilege:          "view",
			Resource:           "any",
			CreatedBy:          uid.ID(778),
			OrganizationMember: models.OrganizationMember{OrganizationID: otherOrg.ID},
		}
		createGrants(t, tx.WithOrgID(otherOrg.ID), otherOrgGrant)

		connectorUser := InfraConnectorIdentity(db)

		connector, err := GetGrant(tx, GetGrantOptions{
			BySubject:   models.NewSubjectForUser(connectorUser.ID),
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
			actual, err := ListGrants(tx, ListGrantsOptions{BySubject: models.NewSubjectForUser(71234)})
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
				BySubject:                  models.NewSubjectForUser(userID),
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

func TestCountAllGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createGrants(t, db,
			&models.Grant{Subject: models.NewSubjectForUser(2002), Privilege: "priv", Resource: "res1"},
			&models.Grant{Subject: models.NewSubjectForUser(2002), Privilege: "priv", Resource: "res2"},
			&models.Grant{Subject: models.NewSubjectForUser(2002), Privilege: "priv", Resource: "res3"})

		actual, err := CountAllGrants(db)
		assert.NilError(t, err)
		assert.Equal(t, actual, int64(4)) // 3 + connector grant
	})
}
