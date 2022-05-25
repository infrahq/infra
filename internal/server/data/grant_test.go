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
			Subject:   "i:1234567",
			Privilege: "view",
			Resource:  "infra",
		}
		g2 := g

		err := CreateGrant(db, &g)
		assert.NilError(t, err)

		err = CreateGrant(db, &g2)
		assert.NilError(t, err)

		grants, err := ListGrants(db, BySubject("i:1234567"), ByResource("infra"))
		assert.NilError(t, err)
		assert.Assert(t, is.Len(grants, 1))
	})
}
