package access

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestListIdentities(t *testing.T) {
	// create the identity
	c, db, infraProvider := setupAccessTestContext(t)

	activeIdentity := &models.Identity{Name: "active-list-hide-id"}

	err := data.CreateIdentity(db, activeIdentity)
	assert.NilError(t, err)

	_, err = data.CreateProviderUser(db, infraProvider, activeIdentity)
	assert.NilError(t, err)

	unlinkedIdentity := &models.Identity{Name: "unlinked-list-hide-id"}

	err = data.CreateIdentity(db, unlinkedIdentity)
	assert.NilError(t, err)

	// test fetch all identities
	ids, err := ListIdentities(c, "", nil)
	assert.NilError(t, err)

	assert.Equal(t, len(ids), 4) // the two identities created, the admin one used to call these access functions, and the internal connector identity
	// make sure both names are seen
	returnedNames := make(map[string]bool)
	for _, id := range ids {
		returnedNames[id.Name] = true
	}
	assert.Equal(t, returnedNames["admin@example.com"], true)
	assert.Equal(t, returnedNames["active-list-hide-id"], true)
	assert.Equal(t, returnedNames["unlinked-list-hide-id"], true)
}

func TestDeleteIdentityCleansUpResources(t *testing.T) {
	// create the identity
	c, db, infraProvider := setupAccessTestContext(t)

	identity := &models.Identity{Name: "to-be-deleted"}

	err := data.CreateIdentity(db, identity)
	assert.NilError(t, err)

	// create some resources for this identity

	keyID := generate.MathRandom(models.AccessKeyKeyLength)
	_, err = data.CreateAccessKey(db, &models.AccessKey{KeyID: keyID, IssuedFor: identity.ID, ProviderID: infraProvider.ID})
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
