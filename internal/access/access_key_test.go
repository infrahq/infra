package access

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestAccessKeys_SelfManagement(t *testing.T) {
	db := setupDB(t)

	org := &models.Organization{Name: "joe's jackets", Domain: "joes-jackets"}
	err := data.CreateOrganization(db, org)
	assert.NilError(t, err)
	db.OrganizationID()

	user := &models.Identity{Name: "joe@example.com", OrganizationMember: models.OrganizationMember{OrganizationID: org.ID}}
	err = data.CreateIdentity(db, user)
	assert.NilError(t, err)

	c, _ := loginAs(&data.Transaction{DB: db.DB}, user, org)

	t.Run("can manage my own keys", func(t *testing.T) {
		key := &models.AccessKey{
			Name:               "foo key",
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			IssuedFor:          user.ID,
			ExpiresAt:          time.Now().Add(1 * time.Minute),
		}
		_, err = CreateAccessKey(c, key)
		assert.NilError(t, err)

		err = DeleteAccessKey(c, key.ID, "")
		assert.NilError(t, err)
	})

	t.Run("can list my own keys", func(t *testing.T) {
		_, err := ListAccessKeys(c, user.ID, "", true, &data.Pagination{})
		assert.NilError(t, err)
	})

	t.Run("can list my own key by name", func(t *testing.T) {
		key := &models.AccessKey{
			Name:               "foo2 key",
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			IssuedFor:          user.ID,
			ExpiresAt:          time.Now().Add(1 * time.Minute),
		}
		_, err = CreateAccessKey(c, key)
		assert.NilError(t, err)

		keys, err := ListAccessKeys(c, user.ID, key.Name, false, nil)
		assert.NilError(t, err)
		assert.Assert(t, len(keys) >= 1)
	})
}
