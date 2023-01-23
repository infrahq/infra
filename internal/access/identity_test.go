package access

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestListIdentities(t *testing.T) {
	rCtx := setupAccessTestContext(t)
	db := rCtx.DBTxn
	infraProvider := data.InfraProvider(db)

	activeIdentity := &models.Identity{Name: "active-list-hide-id"}

	err := data.CreateIdentity(db, activeIdentity)
	assert.NilError(t, err)

	_, err = data.CreateProviderUser(db, infraProvider, activeIdentity)
	assert.NilError(t, err)

	unlinkedIdentity := &models.Identity{Name: "unlinked-list-hide-id"}

	err = data.CreateIdentity(db, unlinkedIdentity)
	assert.NilError(t, err)

	// test fetch all identities
	ids, err := ListIdentities(rCtx, data.ListIdentityOptions{})
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
	rCtx := setupAccessTestContext(t)
	db := rCtx.DBTxn
	infraProvider := data.InfraProvider(db)

	identity := &models.Identity{Name: "to-be-deleted"}

	err := data.CreateIdentity(db, identity)
	assert.NilError(t, err)

	_, err = data.CreateProviderUser(db, infraProvider, identity)
	assert.NilError(t, err)

	// create some resources for this identity

	keyID := generate.MathRandom(models.AccessKeyKeyLength, generate.CharsetAlphaNumeric)
	_, err = data.CreateAccessKey(db, &models.AccessKey{KeyID: keyID, IssuedForID: identity.ID, ProviderID: infraProvider.ID})
	assert.NilError(t, err)

	creds := &models.Credential{
		IdentityID:   identity.ID,
		PasswordHash: []byte("some password"),
	}
	err = data.CreateCredential(db, creds)
	assert.NilError(t, err)

	grantInfra := &models.Grant{
		Subject:         models.NewSubjectForUser(identity.ID),
		DestinationName: models.GrantDestinationInfra,
		Privilege:       "admin",
	}
	err = data.CreateGrant(db, grantInfra)
	assert.NilError(t, err)

	grantDestination := &models.Grant{
		Subject:         models.NewSubjectForUser(identity.ID),
		DestinationName: "example",
		Privilege:       "cluster-admin",
	}
	err = data.CreateGrant(db, grantDestination)
	assert.NilError(t, err)

	group := &models.Group{Name: "Group"}
	err = data.CreateGroup(db, group)
	assert.NilError(t, err)
	err = data.AddUsersToGroup(db, group.ID, []uid.ID{identity.ID})
	assert.NilError(t, err)

	// delete the identity, and make sure all their resources are gone
	err = DeleteIdentity(rCtx, identity.ID)
	assert.NilError(t, err)

	_, err = data.GetIdentity(db, data.GetIdentityOptions{ByID: identity.ID})
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetProviderUser(db, infraProvider.ID, identity.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetAccessKeyByKeyID(db, keyID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetCredentialByUserID(db, identity.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	grants, err := data.ListGrants(db, data.ListGrantsOptions{BySubject: models.NewSubjectForUser(identity.ID)})
	assert.NilError(t, err)
	assert.Equal(t, len(grants), 0)

	group, err = data.GetGroup(db, data.GetGroupOptions{ByID: group.ID})
	assert.NilError(t, err)
	assert.Equal(t, group.TotalUsers, 0)
}
