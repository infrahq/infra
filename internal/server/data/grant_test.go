package data

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/models"
)

func TestDuplicateGrant(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		g := models.Grant{
			Model:     models.Model{ID: 1},
			Subject:   "i:1234567",
			Privilege: "view",
			Resource:  "infra",
		}
		g2 := models.Grant{
			Model:     models.Model{ID: 2},
			Subject:   "i:1234567",
			Privilege: "view",
			Resource:  "infra",
		}

		err := CreateGrant(db, &g)
		assert.NilError(t, err)

		err = CreateGrant(db, &g2)
		assert.ErrorContains(t, err, "already exists")

		grants, err := ListGrants(db, nil, BySubject("i:1234567"), ByResource("infra"))
		assert.NilError(t, err)
		assert.Assert(t, is.Len(grants, 1))

		g3 := models.Grant{
			Model:     models.Model{ID: 3},
			Subject:   "i:1234567",
			Privilege: "edit",
			Resource:  "infra",
		}
		// check that unique constraint needs all three fields

		err = CreateGrant(db, &g3)
		assert.NilError(t, err)
	})
}
