package data

import (
	"testing"

	"github.com/infrahq/infra/internal/server/models"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestDuplicateGrant(t *testing.T) {
	db := setup(t)
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
}
