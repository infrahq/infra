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
