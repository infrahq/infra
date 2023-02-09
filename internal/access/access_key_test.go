package access

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: move test coverage to API handler
func TestAccessKeys_SelfManagement(t *testing.T) {
	db := setupDB(t)

	org := &models.Organization{Name: "joe's jackets", Domain: "joes-jackets"}
	err := data.CreateOrganization(db, org)
	assert.NilError(t, err)

	user := &models.Identity{Name: "joe@example.com", OrganizationMember: models.OrganizationMember{OrganizationID: org.ID}}
	err = data.CreateIdentity(db, user)
	assert.NilError(t, err)

	tx := txnForTestCase(t, db).WithOrgID(org.ID)
	rCtx := RequestContext{
		DBTxn:         tx,
		Authenticated: Authenticated{User: user, Organization: org},
	}

	t.Run("can manage my own keys", func(t *testing.T) {
		key := &models.AccessKey{
			Name:               "foo key",
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			IssuedForID:        user.ID,
			ExpiresAt:          time.Now().Add(1 * time.Minute),
		}
		_, err = CreateAccessKey(rCtx, key)
		assert.NilError(t, err)

		r := rCtx // shallow copy
		r.Authenticated.AccessKey = &models.AccessKey{}
		err = DeleteAccessKey(r, key.ID, "")
		assert.NilError(t, err)
	})

	t.Run("can list my own keys", func(t *testing.T) {
		_, err := ListAccessKeys(rCtx, user.ID, "", true, &data.Pagination{})
		assert.NilError(t, err)
	})

	t.Run("can list my own key by name", func(t *testing.T) {
		key := &models.AccessKey{
			Name:               "foo2 key",
			OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
			IssuedForID:        user.ID,
			ExpiresAt:          time.Now().Add(1 * time.Minute),
		}
		_, err = CreateAccessKey(rCtx, key)
		assert.NilError(t, err)

		keys, err := ListAccessKeys(rCtx, user.ID, key.Name, false, nil)
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

	tx := txnForTestCase(t, db).WithOrgID(org.ID)

	t.Run("admin role", func(t *testing.T) {
		user := &models.Identity{Name: "admin@example.com", OrganizationMember: orgMember}
		err = data.CreateIdentity(db, user)
		assert.NilError(t, err)

		err = data.CreateGrant(db, &models.Grant{Subject: models.NewSubjectForUser(user.ID), Privilege: "admin", DestinationName: models.GrantDestinationInfra, OrganizationMember: orgMember})
		assert.NilError(t, err)

		key := &models.AccessKey{Name: "admin key", IssuedForID: user.ID, ExpiresAt: time.Now().Add(1 * time.Minute), OrganizationMember: orgMember}
		_, err = data.CreateAccessKey(tx, key)
		assert.NilError(t, err)

		rCtx := RequestContext{
			DBTxn:         tx,
			Authenticated: Authenticated{User: user, Organization: org, AccessKey: key},
		}

		t.Run("can create access key for self", func(t *testing.T) {
			key := &models.AccessKey{
				Name:               "a key",
				OrganizationMember: orgMember,
				IssuedForID:        user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}
			_, err := CreateAccessKey(rCtx, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})

		t.Run("can create access key for another user", func(t *testing.T) {
			user := &models.Identity{Name: "bob@example.com", OrganizationMember: orgMember}
			err = data.CreateIdentity(db, user)
			assert.NilError(t, err)

			key := &models.AccessKey{
				Name:               "b key",
				OrganizationMember: orgMember,
				IssuedForID:        user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}
			_, err := CreateAccessKey(rCtx, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})

		t.Run("can create connector access key", func(t *testing.T) {
			connector := data.InfraConnectorIdentity(tx)
			key := &models.AccessKey{
				Name:               "c key",
				OrganizationMember: orgMember,
				IssuedForID:        connector.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}

			_, err := CreateAccessKey(rCtx, key)
			assert.NilError(t, err)
		})
	})

	t.Run("non admin role", func(t *testing.T) {
		user := &models.Identity{Name: "user@example.com", OrganizationMember: orgMember}
		err = data.CreateIdentity(db, user)
		assert.NilError(t, err)

		key := &models.AccessKey{Name: "user key", IssuedForID: user.ID, ExpiresAt: time.Now().Add(1 * time.Minute), OrganizationMember: orgMember}
		_, err = data.CreateAccessKey(tx, key)
		assert.NilError(t, err)

		rCtx := RequestContext{
			DBTxn:         tx,
			Authenticated: Authenticated{User: user, Organization: org, AccessKey: key},
		}

		t.Run("cannot create access key", func(t *testing.T) {
			key := &models.AccessKey{
				Name:               "d key",
				OrganizationMember: orgMember,
				IssuedForID:        user.ID,
				ExpiresAt:          time.Now().Add(1 * time.Minute),
			}

			_, err := CreateAccessKey(rCtx, key)
			assert.ErrorContains(t, err, "cannot use an access key to create other access keys")
		})
	})
}
