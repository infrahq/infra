package data

import (
	"testing"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/stretchr/testify/require"
)

func TestDuplicateGrant(t *testing.T) {
	db := setup(t)
	g := models.Grant{
		Identity:  "u:1234567",
		Privilege: "view",
		Resource:  "infra",
	}
	g2 := g

	err := CreateGrant(db, &g)
	require.NoError(t, err)

	err = CreateGrant(db, &g2)
	require.NoError(t, err)

	grants, err := ListGrants(db, ByIdentity("u:1234567"), ByResource("infra"))
	require.Len(t, grants, 1)
}
