package access

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestDeleteIdentityCleansUpResources(t *testing.T) {
	// create the identity
	c, db, _ := setupAccessTestContext(t)

	identity := &models.Identity{
		Kind: models.UserKind,
		Name: "to-be-deleted",
	}

	err := data.CreateIdentity(db, identity)
	assert.NilError(t, err)

	// create some resources for this identity

	keyID := generate.MathRandom(models.AccessKeyKeyLength)
	_, err = data.CreateAccessKey(db, &models.AccessKey{KeyID: keyID, IssuedFor: identity.ID})
	assert.NilError(t, err)

	creds := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: []byte("some password"),
	}
	err = data.CreateCredential(db, creds)
	assert.NilError(t, err)

	grantInfra := &models.Grant{
		Subject:   identity.PolyID(),
		Resource:  "infra",
		Privilege: "admin",
	}
	err = data.CreateGrant(db, grantInfra)
	assert.NilError(t, err)

	grantDestination := &models.Grant{
		Subject:   identity.PolyID(),
		Resource:  "kubernetes.example",
		Privilege: "cluster-admin",
	}
	err = data.CreateGrant(db, grantDestination)
	assert.NilError(t, err)

	// delete the identity, and make sure all their resources are gone
	err = DeleteIdentity(c, identity.ID)
	assert.NilError(t, err)

	_, err = data.GetIdentity(db, data.ByID(identity.ID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetAccessKey(db, data.ByKeyID(keyID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetCredential(db, data.ByIdentityID(identity.ID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	grants, err := data.ListGrants(db, data.BySubject(identity.PolyID()))
	assert.NilError(t, err)
	assert.Equal(t, len(grants), 0)
}
