package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
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
			listener, err := ListenForNotify(ctx, db, ListenForNotifyOptions{
				GrantsByDestination: "match",
				OrgID:               defaultOrganizationID,
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
		t.Run("notify", func(t *testing.T) {
			g := models.Grant{
				Subject:   "i:1234567",
				Privilege: "view",
				Resource:  "match.a.resource",
			}
			assert.NilError(t, CreateGrant(db, &g))

			ctx := context.Background()
			listener, err := ListenForNotify(ctx, db, ListenForNotifyOptions{
				GrantsByDestination: "match",
				OrgID:               defaultOrganizationID,
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

func TestUpdateGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		var startUpdateIndex int64 = 10001

		t.Run("success add", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			addGrants := []*models.Grant{
				{Subject: "i:7654321", Privilege: "view", Resource: "foo", CreatedBy: uid.ID(1091)},
				{Subject: "i:1234567", Privilege: "admin", Resource: "foo", CreatedBy: uid.ID(1091)},
			}
			rmGrants := []*models.Grant{}

			err := UpdateGrants(tx, addGrants, rmGrants)
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
					Subject:            "i:7654321",
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
					Subject:            "i:1234567",
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
		})
		t.Run("success delete", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			grant1 := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "foo"}
			grant2 := &models.Grant{Subject: "i:any1", Privilege: "edit", Resource: "foo"}
			toKeep := &models.Grant{Subject: "i:any2", Privilege: "view", Resource: "foo"}
			createGrants(t, tx, grant1, grant2, toKeep)

			otherOrgGrant := &models.Grant{Subject: "i:any1", Privilege: "view", Resource: "foo"}
			createGrants(t, tx.WithOrgID(otherOrg.ID), otherOrgGrant)

			addGrants := []*models.Grant{}
			rmGrants := []*models.Grant{grant1, grant2}

			err := UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			// check for idempotency
			err = UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err = ListGrants(tx, ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			// other org still has the grant
			actual, err = ListGrants(tx.WithOrgID(otherOrg.ID), ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 1)
		})
		t.Run("success create and delete", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			addGrants := []*models.Grant{
				{Subject: "i:7654321", Privilege: "view", Resource: "foo", CreatedBy: uid.ID(1091)},
				{Subject: "i:1234567", Privilege: "admin", Resource: "foo", CreatedBy: uid.ID(1091)},
			}
			rmGrants := []*models.Grant{
				{Subject: "i:7654321", Privilege: "view", Resource: "foo", CreatedBy: uid.ID(1091)},
				{Subject: "i:any1", Privilege: "view", Resource: "foo"},
				{Subject: "i:any1", Privilege: "edit", Resource: "foo"},
			}

			err := UpdateGrants(tx, addGrants, rmGrants)
			assert.NilError(t, err)

			actual, err := ListGrants(tx, ListGrantsOptions{ByDestination: "foo"})
			assert.NilError(t, err)
			expected := []models.Grant{
				{
					Model: models.Model{
						ID:        uid.ID(999),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					OrganizationMember: models.OrganizationMember{OrganizationID: defaultOrganizationID},
					Subject:            "i:1234567",
					Privilege:          "admin",
					Resource:           "foo",
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

func TestListenForGrantsNotify(t *testing.T) {
	type operation struct {
		name        string
		run         func(t *testing.T, tx WriteTxn)
		expectMatch bool
	}
	type testCase struct {
		name string
		opts ListenForNotifyOptions
		ops  []operation
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runDBTests(t, func(t *testing.T, db *DB) {
		mainOrg := &models.Organization{Name: "Main", Domain: "main.example.org"}
		assert.NilError(t, CreateOrganization(db, mainOrg))

		otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

		run := func(t *testing.T, tc testCase) {
			listener, err := ListenForNotify(ctx, db, tc.opts)
			assert.NilError(t, err)

			chResult := make(chan struct{})
			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				for {
					err := listener.WaitForNotification(ctx)
					switch {
					case errors.Is(err, context.Canceled):
						return nil
					case err != nil:
						return err
					}
					select {
					case chResult <- struct{}{}:
					case <-ctx.Done():
						return nil
					}
				}
			})

			for _, op := range tc.ops {
				t.Run(op.name, func(t *testing.T) {
					tx, err := db.Begin(ctx, nil)
					assert.NilError(t, err)
					tx = tx.WithOrgID(mainOrg.ID)
					op.run(t, tx)
					assert.NilError(t, tx.Commit())

					if op.expectMatch {
						isNotBlocked(t, chResult)
						return
					}
					isBlocked(t, chResult)
				})
			}

			cancel()
			assert.NilError(t, g.Wait())
		}

		testcases := []testCase{
			{
				name: "by destination",
				opts: ListenForNotifyOptions{
					GrantsByDestination: "mydest",
					OrgID:               mainOrg.ID,
				},
				ops: []operation{
					{
						name: "grant resource matches exactly",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   "i:geo",
								Resource:  "mydest",
								Privilege: "view",
							})
							assert.NilError(t, err)
						},
						expectMatch: true,
					},
					{
						name: "grant resource does not match",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   "i:geo",
								Resource:  "otherdest",
								Privilege: "mydest",
							})
							assert.NilError(t, err)
						},
					},
					{
						name: "grant resource prefix match",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								Subject:   "i:geo",
								Resource:  "mydest.also.ns1",
								Privilege: "admin",
							})
							assert.NilError(t, err)
						},
						expectMatch: true,
					},
					{
						name: "different org",
						run: func(t *testing.T, tx WriteTxn) {
							err := CreateGrant(tx, &models.Grant{
								OrganizationMember: models.OrganizationMember{
									OrganizationID: otherOrg.ID,
								},
								Subject:   "i:geo",
								Resource:  "mydest",
								Privilege: "admin",
							})
							assert.NilError(t, err)
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				run(t, tc)
			})
		}
	})
}

func isBlocked[T any](t *testing.T, ch chan T) {
	t.Helper()
	select {
	case item := <-ch:
		t.Fatalf("expected operation to be blocked, but it returned: %v", item)
	case <-time.After(200 * time.Millisecond):
	}
}

func isNotBlocked[T any](t *testing.T, ch chan T) (result T) {
	t.Helper()
	timeout := 100 * time.Millisecond
	select {
	case item := <-ch:
		return item
	case <-time.After(timeout):
		t.Fatalf("expected operation to not block, timeout after: %v", timeout)
		return result
	}
}

func TestCountAllGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createGrants(t, db,
			&models.Grant{Subject: "sub", Privilege: "priv", Resource: "res1"},
			&models.Grant{Subject: "sub", Privilege: "priv", Resource: "res2"},
			&models.Grant{Subject: "sub", Privilege: "priv", Resource: "res3"})

		actual, err := CountAllGrants(db)
		assert.NilError(t, err)
		assert.Equal(t, actual, int64(4)) // 3 + connector grant
	})
}
