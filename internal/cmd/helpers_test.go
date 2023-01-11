package cmd

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func createGrants(t *testing.T, tx data.WriteTxn, grants ...api.GrantRequest) {
	t.Helper()
	for i, g := range grants {
		var subject uid.PolymorphicID
		switch {
		case g.User != 0:
			subject = uid.NewIdentityPolymorphicID(g.User)
		case g.Group != 0:
			subject = uid.NewGroupPolymorphicID(g.Group)
		case g.UserName != "":
			u, err := data.GetIdentity(tx, data.GetIdentityOptions{ByName: g.UserName})
			assert.NilError(t, err, "grant %v", i)
			subject = uid.NewIdentityPolymorphicID(u.ID)
		case g.GroupName != "":
			group, err := data.GetGroup(tx, data.GetGroupOptions{ByName: g.GroupName})
			assert.NilError(t, err, "grant %v", i)
			subject = uid.NewGroupPolymorphicID(group.ID)
		}

		err := data.CreateGrant(tx, &models.Grant{
			Subject:   subject,
			Resource:  g.Resource,
			Privilege: g.Privilege,
		})
		assert.NilError(t, err, "grant %v", i)
	}
}
