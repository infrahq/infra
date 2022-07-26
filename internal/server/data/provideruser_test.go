package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

// mockOIDC is a mock oidc identity provider
type mockOIDCImplementation struct {
	UserEmailResp  string
	UserGroupsResp []string
}

func (m *mockOIDCImplementation) Validate(_ context.Context) error {
	return nil
}

func (m *mockOIDCImplementation) AuthServerInfo(_ context.Context) (*providers.AuthServerInfo, error) {
	return &providers.AuthServerInfo{AuthURL: "example.com/v1/auth", ScopesSupported: []string{"openid", "email"}}, nil
}

func (m *mockOIDCImplementation) ExchangeAuthCodeForProviderTokens(_ context.Context, _ string) (acc, ref string, exp time.Time, email string, err error) {
	return "acc", "ref", exp, m.UserEmailResp, nil
}

func (m *mockOIDCImplementation) RefreshAccessToken(_ context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	if providerUser.ExpiresAt.Before(time.Now()) {
		exp := time.Now().Add(1 * time.Hour)
		return "new-acc-token", &exp, nil
	}
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *mockOIDCImplementation) GetUserInfo(_ context.Context, providerUser *models.ProviderUser) (*providers.UserInfoClaims, error) {
	return &providers.UserInfoClaims{Email: m.UserEmailResp, Groups: m.UserGroupsResp}, nil
}

var cmpEncryptedAtRestNotZero = cmp.Comparer(func(x, y models.EncryptedAtRest) bool {
	return x != "" && y != ""
})

func TestSyncProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		provider := &models.Provider{
			Name: "mockta",
			Kind: models.ProviderKindOkta,
		}

		err := CreateProvider(db, provider)
		assert.NilError(t, err)

		tests := []struct {
			name              string
			setupProviderUser func(t *testing.T) *models.Identity
			oidcClient        providers.OIDCClient
			verifyFunc        func(t *testing.T, err error, user *models.Identity)
		}{
			{
				name: "invalid/expired access token is updated",
				setupProviderUser: func(t *testing.T) *models.Identity {
					user := &models.Identity{
						Name: "hello@example.com",
					}

					err = CreateIdentity(db, user)
					assert.NilError(t, err)

					pu := &models.ProviderUser{
						ProviderID: provider.ID,
						IdentityID: user.ID,

						Email:        user.Name,
						RedirectURL:  "http://example.com",
						AccessToken:  models.EncryptedAtRest("aaa"),
						RefreshToken: models.EncryptedAtRest("bbb"),
						ExpiresAt:    time.Now().UTC().Add(-5 * time.Minute),
						LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
					}

					err = UpdateProviderUser(db, pu)
					assert.NilError(t, err)

					return user
				},
				oidcClient: &mockOIDCImplementation{
					UserEmailResp:  "hello@example.com",
					UserGroupsResp: []string{"Everyone", "Developers"},
				},
				verifyFunc: func(t *testing.T, err error, user *models.Identity) {
					assert.NilError(t, err)

					pu, err := GetProviderUser(db, provider.ID, user.ID)
					assert.NilError(t, err)

					expected := models.ProviderUser{
						Model:        pu.Model, // not relevant
						Email:        "hello@example.com",
						Groups:       models.CommaSeparatedStrings{"Everyone", "Developers"},
						ProviderID:   provider.ID,
						IdentityID:   user.ID,
						RedirectURL:  "http://example.com",
						RefreshToken: "bbb",
						AccessToken:  "any-access-token",
						ExpiresAt:    time.Now().Add(time.Hour).UTC(),
						LastUpdate:   time.Now().UTC(),
					}

					cmpProviderUser := cmp.Options{
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "ExpiresAt"),
							opt.TimeWithThreshold(20*time.Second)),
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "LastUpdate"),
							opt.TimeWithThreshold(20*time.Second)),
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "AccessToken"),
							cmpEncryptedAtRestNotZero),
					}

					assert.DeepEqual(t, *pu, expected, cmpProviderUser)
				},
			},
			{
				name: "groups are updated to match user info",
				setupProviderUser: func(t *testing.T) *models.Identity {
					user := &models.Identity{
						Name: "sync@example.com",
					}

					err = CreateIdentity(db, user)
					assert.NilError(t, err)

					pu := &models.ProviderUser{
						ProviderID: provider.ID,
						IdentityID: user.ID,

						Email:        user.Name,
						RedirectURL:  "http://example.com",
						AccessToken:  models.EncryptedAtRest("aaa"),
						RefreshToken: models.EncryptedAtRest("bbb"),
						ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
						LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
					}

					err = UpdateProviderUser(db, pu)
					assert.NilError(t, err)

					return user
				},
				oidcClient: &mockOIDCImplementation{
					UserEmailResp:  "sync@example.com",
					UserGroupsResp: []string{"Everyone", "Developers"},
				},
				verifyFunc: func(t *testing.T, err error, user *models.Identity) {
					assert.NilError(t, err)

					pu, err := GetProviderUser(db, provider.ID, user.ID)
					assert.NilError(t, err)

					expected := models.ProviderUser{
						Model:        pu.Model, // not relevant
						Email:        "sync@example.com",
						Groups:       models.CommaSeparatedStrings{"Everyone", "Developers"},
						ProviderID:   provider.ID,
						IdentityID:   user.ID,
						RedirectURL:  "http://example.com",
						RefreshToken: "bbb",
						AccessToken:  "any-access-token",
						ExpiresAt:    time.Now().Add(5 * time.Minute).UTC(),
						LastUpdate:   time.Now().UTC(),
					}

					cmpProviderUser := cmp.Options{
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "ExpiresAt"),
							opt.TimeWithThreshold(20*time.Second)),
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "LastUpdate"),
							opt.TimeWithThreshold(20*time.Second)),
						cmp.FilterPath(
							opt.PathField(models.ProviderUser{}, "AccessToken"),
							cmpEncryptedAtRestNotZero),
					}

					assert.DeepEqual(t, *pu, expected, cmpProviderUser)

					assert.Assert(t, len(pu.Groups) == 2)

					puGroups := make(map[string]bool)
					for _, g := range pu.Groups {
						puGroups[g] = true
					}

					assert.Assert(t, puGroups["Everyone"])
					assert.Assert(t, puGroups["Developers"])

					// check that the direct user-to-group relation was updated
					storedGroups, err := ListGroups(db, nil, ByGroupMember(pu.IdentityID))
					assert.NilError(t, err)

					userGroups := make(map[string]bool)
					for _, g := range storedGroups {
						userGroups[g.Name] = true
					}

					assert.Assert(t, userGroups["Everyone"])
					assert.Assert(t, userGroups["Developers"])
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				oidc := test.oidcClient
				user := test.setupProviderUser(t)
				err = SyncProviderUser(context.Background(), db, user, provider, oidc)
				test.verifyFunc(t, err, user)
			})
		}
	})
}
