package authn

import (
	"context"
	"testing"
	"time"

	"github.com/ssoroka/slice"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
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
	// never update
	return string(providerUser.AccessToken), &providerUser.ExpiresAt, nil
}

func (m *mockOIDCImplementation) GetUserInfo(_ context.Context, providerUser *models.ProviderUser) (*providers.UserInfoClaims, error) {
	return &providers.UserInfoClaims{Email: m.UserEmailResp, Groups: m.UserGroupsResp}, nil
}

func TestOIDCAuthenticate(t *testing.T) {
	// setup
	db := setupDB(t)

	mocktaProvider := models.Provider{Name: "mockta", Kind: models.ProviderKindOkta}
	err := data.CreateProvider(db, &mocktaProvider)
	assert.NilError(t, err)

	oidc := &mockOIDCImplementation{
		UserEmailResp:  "bruce@example.com",
		UserGroupsResp: []string{"Everyone", "developers"},
	}

	t.Run("invalid provider", func(t *testing.T) {
		unknownProviderOIDCAuthn := NewOIDCAuthentication(uid.New(), "localhost:8031", "1234", oidc)
		_, err := unknownProviderOIDCAuthn.Authenticate(context.Background(), db, time.Now().Add(1*time.Minute))

		assert.ErrorIs(t, err, internal.ErrNotFound)
	})

	t.Run("successful authentication", func(t *testing.T) {
		oidcAuthn := NewOIDCAuthentication(mocktaProvider.ID, "localhost:8031", "1234", oidc)
		authnIdentity, err := oidcAuthn.Authenticate(context.Background(), db, time.Now().Add(1*time.Minute))

		assert.NilError(t, err)
		// user should be created
		assert.Equal(t, authnIdentity.Identity.Name, "bruce@example.com")

		groups := make(map[string]bool)
		for _, g := range authnIdentity.Identity.Groups {
			groups[g.Name] = true
		}
		assert.Assert(t, len(authnIdentity.Identity.Groups) == 2)
		assert.Equal(t, groups["Everyone"], true)
		assert.Equal(t, groups["developers"], true)

		assert.Equal(t, authnIdentity.Provider.ID, mocktaProvider.ID)
	})
}

func TestExchangeAuthCodeForProviderTokens(t *testing.T) {
	sessionExpiry := time.Now().Add(5 * time.Minute)

	type testCase struct {
		setup    func(t *testing.T, db data.GormTxn) providers.OIDCClient
		expected func(t *testing.T, authnIdentity AuthenticatedIdentity)
	}

	testCases := map[string]testCase{
		"NewUserNewGroups": {
			setup: func(t *testing.T, db data.GormTxn) providers.OIDCClient {
				return &mockOIDCImplementation{
					UserEmailResp:  "newusernewgroups@example.com",
					UserGroupsResp: []string{"Everyone", "developers"},
				}
			},
			expected: func(t *testing.T, a AuthenticatedIdentity) {
				assert.Equal(t, "newusernewgroups@example.com", a.Identity.Name)
				assert.Equal(t, "mockoidc", a.Provider.Name)
				assert.Assert(t, a.SessionExpiry.Equal(sessionExpiry))
			},
		},
		"NewUserExistingGroups": {
			setup: func(t *testing.T, db data.GormTxn) providers.OIDCClient {
				existingGroup1 := &models.Group{Name: "existing1"}
				existingGroup2 := &models.Group{Name: "existing2"}

				err := data.CreateGroup(db, existingGroup1)
				assert.NilError(t, err)

				err = data.CreateGroup(db, existingGroup2)
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "newuserexistinggroups@example.com",
					UserGroupsResp: []string{"existing1", "existing2"},
				}
			},
			expected: func(t *testing.T, a AuthenticatedIdentity) {
				assert.Equal(t, "newuserexistinggroups@example.com", a.Identity.Name)
				assert.Equal(t, "mockoidc", a.Provider.Name)
				assert.Assert(t, a.SessionExpiry.Equal(sessionExpiry))

				assert.Assert(t, is.Len(a.Identity.Groups, 2))

				var groupNames []string
				for _, g := range a.Identity.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existing1"))
				assert.Assert(t, is.Contains(groupNames, "existing2"))
			},
		},
		"ExistingUserNewGroups": {
			setup: func(t *testing.T, db data.GormTxn) providers.OIDCClient {
				err := data.CreateIdentity(db, &models.Identity{Name: "existingusernewgroups@example.com"})
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existingusernewgroups@example.com",
					UserGroupsResp: []string{"existingusernewgroups1", "existingusernewgroups2"},
				}
			},
			expected: func(t *testing.T, a AuthenticatedIdentity) {
				assert.Equal(t, "existingusernewgroups@example.com", a.Identity.Name)
				assert.Equal(t, "mockoidc", a.Provider.Name)
				assert.Assert(t, a.SessionExpiry.Equal(sessionExpiry))

				assert.Assert(t, is.Len(a.Identity.Groups, 2))

				var groupNames []string
				for _, g := range a.Identity.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existingusernewgroups1"))
				assert.Assert(t, is.Contains(groupNames, "existingusernewgroups2"))
			},
		},
		"ExistingUserExistingGroups": {
			setup: func(t *testing.T, db data.GormTxn) providers.OIDCClient {
				err := data.CreateIdentity(db, &models.Identity{Name: "existinguserexistinggroups@example.com"})
				assert.NilError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups1"})
				assert.NilError(t, err)

				err = data.CreateGroup(db, &models.Group{Name: "existinguserexistinggroups2"})
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "existinguserexistinggroups@example.com",
					UserGroupsResp: []string{"existinguserexistinggroups1", "existinguserexistinggroups2"},
				}
			},
			expected: func(t *testing.T, a AuthenticatedIdentity) {
				assert.Equal(t, "existinguserexistinggroups@example.com", a.Identity.Name)
				assert.Equal(t, "mockoidc", a.Provider.Name)
				assert.Assert(t, a.SessionExpiry.Equal(sessionExpiry))

				assert.Assert(t, is.Len(a.Identity.Groups, 2))

				var groupNames []string
				for _, g := range a.Identity.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, is.Contains(groupNames, "existinguserexistinggroups1"))
				assert.Assert(t, is.Contains(groupNames, "existinguserexistinggroups2"))
			},
		},
		"ExistingUserGroupsWithNewGroups": {
			setup: func(t *testing.T, db data.GormTxn) providers.OIDCClient {
				user := &models.Identity{Name: "eugwnw@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				for _, name := range []string{"Foo", "existing3"} {
					group := &models.Group{Name: name}
					err = data.CreateGroup(db, group)
					assert.NilError(t, err)
					user.Groups = append(user.Groups, *group)
				}
				assert.NilError(t, err)
				assert.Assert(t, len(user.Groups) == 2)

				err = data.SaveIdentity(db, user)
				assert.NilError(t, err)
				g, err := data.GetGroup(db, data.ByName("Foo"))
				assert.NilError(t, err)
				assert.Assert(t, g != nil)

				user, err = data.GetIdentity(db, data.Preload("Groups"), data.ByID(user.ID))
				assert.NilError(t, err)
				assert.Assert(t, user != nil)
				assert.Assert(t, len(user.Groups) == 2)

				p, err := data.GetProvider(db, data.ByName("mockoidc"))
				assert.NilError(t, err)

				pu, err := data.CreateProviderUser(db, p, user)
				assert.NilError(t, err)

				pu.Groups = []string{"existing3"}
				err = db.GormDB().Save(pu).Error
				assert.NilError(t, err)

				return &mockOIDCImplementation{
					UserEmailResp:  "eugwnw@example.com",
					UserGroupsResp: []string{"existinguserexistinggroups1", "existinguserexistinggroups2"},
				}
			},
			expected: func(t *testing.T, a AuthenticatedIdentity) {
				assert.Equal(t, "eugwnw@example.com", a.Identity.Name)
				assert.Equal(t, "mockoidc", a.Provider.Name)
				assert.Assert(t, a.SessionExpiry.Equal(sessionExpiry))

				assert.Assert(t, len(a.Identity.Groups) == 3)

				var groupNames []string
				for _, g := range a.Identity.Groups {
					groupNames = append(groupNames, g.Name)
				}
				assert.Assert(t, slice.Contains(groupNames, "Foo"))
				assert.Assert(t, slice.Contains(groupNames, "existinguserexistinggroups1"))
				assert.Assert(t, slice.Contains(groupNames, "existinguserexistinggroups2"))
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			db := setupDB(t)

			// setup fake identity provider
			provider := &models.Provider{Name: "mockoidc", URL: "mockOIDC.example.com", Kind: models.ProviderKindOIDC}
			err := data.CreateProvider(db, provider)
			assert.NilError(t, err)

			mockOIDC := tc.setup(t, db)
			loginMethod := NewOIDCAuthentication(provider.ID, "mockOIDC.example.com/redirect", "AAA", mockOIDC)

			a, err := loginMethod.Authenticate(context.Background(), db, sessionExpiry)
			assert.NilError(t, err)
			tc.expected(t, a)

			if err == nil {
				// make sure the associations are still set when you reload the object.
				u, err := data.GetIdentity(db, data.Preload("Groups"), data.ByID(a.Identity.ID))
				assert.NilError(t, err)
				a.Identity = u
				tc.expected(t, a)
			}
		})
	}
}
