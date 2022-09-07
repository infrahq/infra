package data

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

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

			grants, err := ListGrants(tx, nil,
				BySubject("i:1234567"),
				ByResource("infra"))
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
	})
}

func TestDeleteGrants(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		otherOrg := &models.Organization{Name: "other", Domain: "other.example.org"}
		assert.NilError(t, CreateOrganization(db, otherOrg))

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

			actual, err := ListGrants(tx, nil, ByResource("any"))
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
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

			actual, err := ListGrants(tx, nil, ByResource("any"))
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)

			actual, err = ListGrants(tx.WithOrgID(otherOrg.ID), nil, ByResource("any"))
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

			actual, err := ListGrants(tx, nil, ByResource("any"))
			assert.NilError(t, err)
			expected := []models.Grant{
				{Model: models.Model{ID: toKeep1.ID}},
				{Model: models.Model{ID: toKeep2.ID}},
			}
			assert.DeepEqual(t, actual, expected, cmpModelByID)
		})
	})
}

func createGrants(t *testing.T, tx WriteTxn, grants ...*models.Grant) {
	t.Helper()
	for _, grant := range grants {
		assert.NilError(t, CreateGrant(tx, grant))
	}
}
