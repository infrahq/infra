package access

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
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
	ids, err := ListIdentities(c, data.ListIdentityOptions{})
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

	_, err = data.CreateProviderUser(db, infraProvider, identity)
	assert.NilError(t, err)

	// create some resources for this identity

	keyID := generate.MathRandom(models.AccessKeyKeyLength, generate.CharsetAlphaNumeric)
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
		Resource:  "example",
		Privilege: "cluster-admin",
	}
	err = data.CreateGrant(db, grantDestination)
	assert.NilError(t, err)

	group := &models.Group{Name: "Group"}
	err = data.CreateGroup(db, group)
	assert.NilError(t, err)
	err = data.AddUsersToGroup(db, group.ID, []uid.ID{identity.ID})
	assert.NilError(t, err)

	// delete the identity, and make sure all their resources are gone
	err = DeleteIdentity(c, identity.ID)
	assert.NilError(t, err)

	_, err = data.GetIdentity(db, data.GetIdentityOptions{ByID: identity.ID})
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetProviderUser(db, infraProvider.ID, identity.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetAccessKeyByKeyID(db, keyID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetCredentialByUserID(db, identity.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	grants, err := data.ListGrants(db, data.ListGrantsOptions{BySubject: identity.PolyID()})
	assert.NilError(t, err)
	assert.Equal(t, len(grants), 0)

	group, err = data.GetGroup(db, data.GetGroupOptions{ByID: group.ID})
	assert.NilError(t, err)
	assert.Equal(t, group.TotalUsers, 0)
}

// mockOIDC is a fake oidc identity provider
type fakeOIDCImplementation struct {
	UserInfoRevoked bool // when true returns an error fromt the user info endpoint
}

func (m *fakeOIDCImplementation) Validate(_ context.Context) error {
	return nil
}

func (m *fakeOIDCImplementation) AuthServerInfo(_ context.Context) (*providers.AuthServerInfo, error) {
	return &providers.AuthServerInfo{AuthURL: "example.com/v1/auth", ScopesSupported: []string{"openid", "email"}}, nil
}

func (m *fakeOIDCImplementation) ExchangeAuthCodeForProviderTokens(_ context.Context, _ string) (*providers.IdentityProviderAuth, error) {
	return &providers.IdentityProviderAuth{
		AccessToken:       "acc",
		RefreshToken:      "ref",
		AccessTokenExpiry: time.Now().Add(1 * time.Minute),
		Email:             "hello@example.com",
	}, nil
}

func (m *fakeOIDCImplementation) RefreshAccessToken(_ context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	// never update
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *fakeOIDCImplementation) GetUserInfo(_ context.Context, _ *models.ProviderUser) (*providers.UserInfoClaims, error) {
	if m.UserInfoRevoked {
		return nil, fmt.Errorf("user revoked")
	}
	return &providers.UserInfoClaims{}, nil
}

func TestUpdateIdentityInfoFromProvider(t *testing.T) {
	// create the identity
	c, db, infraProvider := setupAccessTestContext(t)

	rCtx := GetRequestContext(c)
	rCtx.Request = &http.Request{}

	provider := &models.Provider{
		Name:         "mockta",
		URL:          "example.com",
		ClientID:     "aaa",
		ClientSecret: "bbb",
		Kind:         models.ProviderKindOIDC,
	}

	err := data.CreateProvider(db, provider)
	assert.NilError(t, err)

	t.Run("a revoked OIDC session revokes access keys created by provider login", func(t *testing.T) {
		_, err = data.CreateProviderUser(db, provider, rCtx.Authenticated.User)
		assert.NilError(t, err)
		oidc := &fakeOIDCImplementation{UserInfoRevoked: true}

		toBeRevoked := &models.AccessKey{IssuedFor: rCtx.Authenticated.User.ID, ProviderID: provider.ID}
		_, err := data.CreateAccessKey(db, toBeRevoked)
		assert.NilError(t, err)
		shouldStayValid := &models.AccessKey{IssuedFor: rCtx.Authenticated.User.ID, ProviderID: infraProvider.ID}
		_, err = data.CreateAccessKey(db, shouldStayValid)
		assert.NilError(t, err)

		rCtx.Authenticated.AccessKey = toBeRevoked

		err = UpdateIdentityInfoFromProvider(rCtx, oidc)
		assert.ErrorContains(t, err, "user revoked")

		_, err = data.GetAccessKeyByKeyID(db, toBeRevoked.KeyID)
		assert.ErrorIs(t, err, internal.ErrNotFound)

		_, err = data.GetAccessKeyByKeyID(db, shouldStayValid.KeyID)
		assert.NilError(t, err)
	})

	t.Run("a valid OIDC session does not result in an error", func(t *testing.T) {
		_, err = data.CreateProviderUser(db, provider, rCtx.Authenticated.User)
		assert.NilError(t, err)
		oidc := &fakeOIDCImplementation{}

		key := &models.AccessKey{IssuedFor: rCtx.Authenticated.User.ID, ProviderID: provider.ID}
		_, err := data.CreateAccessKey(db, key)
		assert.NilError(t, err)

		rCtx.Authenticated.AccessKey = key

		err = UpdateIdentityInfoFromProvider(rCtx, oidc)
		assert.NilError(t, err)
	})
}
