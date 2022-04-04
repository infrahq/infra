package access

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestDeleteIdentityCleansUpResources(t *testing.T) {
	// create the identity
	c, db, provider := setupAccessTestContext(t)

	identity := &models.Identity{
		Kind:       models.UserKind,
		Name:       "to-be-deleted",
		ProviderID: provider.ID,
	}

	err := data.CreateIdentity(db, identity)
	require.NoError(t, err)

	// create some resources for this identity

	keyID := generate.MathRandom(models.AccessKeyKeyLength)
	_, err = data.CreateAccessKey(db, &models.AccessKey{KeyID: keyID, IssuedFor: identity.ID})
	require.NoError(t, err)

	creds := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: []byte("some password"),
	}
	err = data.CreateCredential(db, creds)
	require.NoError(t, err)

	grantInfra := &models.Grant{
		Subject:   identity.PolyID(),
		Resource:  "infra",
		Privilege: "admin",
	}
	err = data.CreateGrant(db, grantInfra)
	require.NoError(t, err)

	grantDestination := &models.Grant{
		Subject:   identity.PolyID(),
		Resource:  "kubernetes.example",
		Privilege: "cluster-admin",
	}
	err = data.CreateGrant(db, grantDestination)
	require.NoError(t, err)

	// delete the identity, and make sure all their resources are gone
	err = DeleteIdentity(c, identity.ID)
	require.NoError(t, err)

	_, err = data.GetIdentity(db, data.ByID(identity.ID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetAccessKey(db, data.ByKeyID(keyID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetCredential(db, data.ByIdentityID(identity.ID))
	assert.ErrorIs(t, err, internal.ErrNotFound)

	grants, err := data.ListGrants(db, data.BySubject(identity.PolyID()))
	require.NoError(t, err)
	assert.Equal(t, len(grants), 0)
}
