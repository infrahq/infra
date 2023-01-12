package cmd

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func createGrants(t *testing.T, tx data.WriteTxn, grants ...api.GrantRequest) {
	t.Helper()
	for i, g := range grants {
		var subject models.Subject
		switch {
		case g.User != 0:
			subject = models.NewSubjectForUser(g.User)
		case g.Group != 0:
			subject = models.NewSubjectForGroup(g.Group)
		case g.UserName != "":
			u, err := data.GetIdentity(tx, data.GetIdentityOptions{ByName: g.UserName})
			assert.NilError(t, err, "grant %v", i)
			subject = models.NewSubjectForUser(u.ID)
		case g.GroupName != "":
			group, err := data.GetGroup(tx, data.GetGroupOptions{ByName: g.GroupName})
			assert.NilError(t, err, "grant %v", i)
			subject = models.NewSubjectForGroup(group.ID)
		}

		err := data.CreateGrant(tx, &models.Grant{
			Subject:   subject,
			Resource:  g.Resource,
			Privilege: g.Privilege,
		})
		assert.NilError(t, err, "grant %v", i)
	}
}
