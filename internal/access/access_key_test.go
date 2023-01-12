package access

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAccessKeys_SelfManagement(t *testing.T) {
	db := setupDB(t)

	org := &models.Organization{Name: "joe's jackets", Domain: "joes-jackets"}
	err := data.CreateOrganization(db, org)
	assert.NilError(t, err)

	user := &models.Identity{Name: "joe@example.com", OrganizationMember: models.OrganizationMember{OrganizationID: org.ID}}
	err = data.CreateIdentity(db, user)
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(nil)
	tx := txnForTestCase(t, db).WithOrgID(org.ID)
	rCtx := RequestContext{
		DBTxn:         tx,
		Authenticated: Authenticated{User: user, Organization: org},
	}
	c.Set(RequestContextKey, rCtx)

	t.Run("can manage my own keys", func(t *testing.T) {
		key := &models.AccessKey{
			Name:               "foo key",
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			IssuedFor:          user.ID,
			ExpiresAt:          time.Now().Add(1 * time.Minute),
		}
		_, err = CreateAccessKey(c, key)
		assert.NilError(t, err)

		r := rCtx // shallow copy
		r.Authenticated.AccessKey = &models.AccessKey{}
		err = DeleteAccessKey(r, key.ID, "")
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

func TestAccessKeys_AccessKeyAuthn(t *testing.T) {
	db := setupDB(t)

	org := &models.Organization{Name: "joe's jackets", Domain: "joes-jackets"}
	err := data.CreateOrganization(db, org)
	assert.NilError(t, err)

	orgMember := models.OrganizationMember{OrganizationID: org.ID}

	c, _ := gin.CreateTestContext(nil)
	tx := txnForTestCase(t, db).WithOrgID(org.ID)

	t.Run("admin role", func(t *testing.T) {
		user := &models.Identity{Name: "admin@example.com", OrganizationMember: orgMember}
		err = data.CreateIdentity(db, user)
		assert.NilError(t, err)

		err = data.CreateGrant(db, &models.Grant{Subject: uid.NewIdentityPolymorphicID(user.ID), Privilege: "admin", Resource: "infra", OrganizationMember: orgMember})
		assert.NilError(t, err)

		key := &models.AccessKey{Name: "admin key", IssuedFor: user.ID, ExpiresAt: time.Now().Add(1 * time.Minute), OrganizationMember: orgMember}
		_, err = data.CreateAccessKey(db, key)
		assert.NilError(t, err)

		rCtx := RequestContext{
			DBTxn:         tx,
			Authenticated: Authenticated{User: user, Organization: org, AccessKey: key},
		}
		c.Set(RequestContextKey, rCtx)

		t.Run("can create access key for self", func(t *testing.T) {
			key := &models.AccessKey{
				Name:               "a key",
				OrganizationMember: orgMember,
				IssuedFor:          user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}
			_, err := CreateAccessKey(c, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})

		t.Run("can create access key for another user", func(t *testing.T) {
			user := &models.Identity{Name: "bob@example.com", OrganizationMember: orgMember}
			err = data.CreateIdentity(db, user)
			assert.NilError(t, err)

			key := &models.AccessKey{
				Name:               "b key",
				OrganizationMember: orgMember,
				IssuedFor:          user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}
			_, err := CreateAccessKey(c, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})

		t.Run("can create connector access key", func(t *testing.T) {
			connector := data.InfraConnectorIdentity(tx)
			key := &models.AccessKey{
				Name:               "c key",
				OrganizationMember: orgMember,
				IssuedFor:          connector.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}

			_, err := CreateAccessKey(c, key)
			assert.NilError(t, err)
		})
	})

	t.Run("non admin role", func(t *testing.T) {
		user := &models.Identity{Name: "user@example.com", OrganizationMember: orgMember}
		err = data.CreateIdentity(db, user)
		assert.NilError(t, err)

		key := &models.AccessKey{Name: "user key", IssuedFor: user.ID, ExpiresAt: time.Now().Add(1 * time.Minute), OrganizationMember: orgMember}
		_, err = data.CreateAccessKey(db, key)
		assert.NilError(t, err)

		rCtx := RequestContext{
			DBTxn:         tx,
			Authenticated: Authenticated{User: user, Organization: org, AccessKey: key},
		}
		c.Set(RequestContextKey, rCtx)

		t.Run("cannot create access key", func(t *testing.T) {
			key := &models.AccessKey{
				Name:               "d key",
				OrganizationMember: orgMember,
				IssuedFor:          user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}

			_, err := CreateAccessKey(c, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})
	})
}
