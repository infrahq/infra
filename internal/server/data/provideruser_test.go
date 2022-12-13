package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/scim2/filter-parser/v2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

// mockOIDC is a mock oidc identity provider
type mockOIDCImplementation struct {
	UserEmailResp  string
	UserGroupsResp []string
	ProviderModel  models.Provider
}

func (m *mockOIDCImplementation) Validate(_ context.Context) error {
	return nil
}

func (m *mockOIDCImplementation) AuthServerInfo(_ context.Context) (*providers.AuthServerInfo, error) {
	return &providers.AuthServerInfo{AuthURL: "example.com/v1/auth", ScopesSupported: []string{"openid", "email"}}, nil
}

func (m *mockOIDCImplementation) ExchangeAuthCodeForProviderTokens(_ context.Context, _ string) (*providers.IdentityProviderAuth, error) {
	return &providers.IdentityProviderAuth{
		AccessToken:       "acc",
		RefreshToken:      "ref",
		AccessTokenExpiry: time.Now().Add(1 * time.Minute),
		Email:             m.UserEmailResp,
	}, nil
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

func (m *mockOIDCImplementation) Provider() *models.Provider {
	return &m.ProviderModel
}

var cmpEncryptedAtRestNotZero = cmp.Comparer(func(x, y models.EncryptedAtRest) bool {
	return x != "" && y != ""
})

func TestSyncProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
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

					pu, err := CreateProviderUser(db, provider, user)
					assert.NilError(t, err)

					pu.RedirectURL = "http://example.com"
					pu.AccessToken = models.EncryptedAtRest("aaa")
					pu.RefreshToken = models.EncryptedAtRest("bbb")
					pu.ExpiresAt = time.Now().UTC().Add(-5 * time.Minute)

					err = UpdateProviderUser(db, pu)
					assert.NilError(t, err)

					return user
				},
				oidcClient: &mockOIDCImplementation{
					UserEmailResp:  "hello@example.com",
					UserGroupsResp: []string{"Everyone", "Developers"},
					ProviderModel:  *provider,
				},
				verifyFunc: func(t *testing.T, err error, user *models.Identity) {
					assert.NilError(t, err)

					pu, err := GetProviderUser(db, provider.ID, user.ID)
					assert.NilError(t, err)

					expected := models.ProviderUser{
						Email:        "hello@example.com",
						Groups:       models.CommaSeparatedStrings{"Everyone", "Developers"},
						ProviderID:   provider.ID,
						IdentityID:   user.ID,
						RedirectURL:  "http://example.com",
						RefreshToken: "bbb",
						AccessToken:  "any-access-token",
						ExpiresAt:    time.Now().Add(time.Hour).UTC(),
						LastUpdate:   time.Now().UTC(),
						Active:       true,
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

					pu, err := CreateProviderUser(db, provider, user)
					assert.NilError(t, err)

					pu.RedirectURL = "http://example.com"
					pu.AccessToken = models.EncryptedAtRest("aaa")
					pu.RefreshToken = models.EncryptedAtRest("bbb")
					pu.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)

					err = UpdateProviderUser(db, pu)
					assert.NilError(t, err)

					return user
				},
				oidcClient: &mockOIDCImplementation{
					UserEmailResp:  "sync@example.com",
					UserGroupsResp: []string{"Everyone", "Developers"},
					ProviderModel:  *provider,
				},
				verifyFunc: func(t *testing.T, err error, user *models.Identity) {
					assert.NilError(t, err)

					pu, err := GetProviderUser(db, provider.ID, user.ID)
					assert.NilError(t, err)

					expected := models.ProviderUser{
						Email:        "sync@example.com",
						Groups:       models.CommaSeparatedStrings{"Everyone", "Developers"},
						ProviderID:   provider.ID,
						IdentityID:   user.ID,
						RedirectURL:  "http://example.com",
						RefreshToken: "bbb",
						AccessToken:  "any-access-token",
						ExpiresAt:    time.Now().Add(5 * time.Minute).UTC(),
						LastUpdate:   time.Now().UTC(),
						Active:       true,
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
					storedGroups, err := ListGroups(db, ListGroupsOptions{ByGroupMember: pu.IdentityID})
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
				err = SyncProviderUser(context.Background(), db, user, oidc)
				test.verifyFunc(t, err, user)
			})
		}
	})
}

func TestDeleteProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		provider := &models.Provider{
			Name: "mockta",
			Kind: models.ProviderKindOkta,
		}

		err := CreateProvider(db, provider)
		assert.NilError(t, err)

		user := &models.Identity{
			Name: "alice@example.com",
		}
		err = CreateIdentity(db, user)
		assert.NilError(t, err)

		_, err = CreateProviderUser(db, provider, user)
		assert.NilError(t, err)

		// check the provider user exists
		_, err = GetProviderUser(db, provider.ID, user.ID)
		assert.NilError(t, err)

		// hard delete the provider user
		err = DeleteProviderUsers(db, DeleteProviderUsersOptions{ByIdentityID: user.ID, ByProviderID: provider.ID})
		assert.NilError(t, err)

		// provider user no longer exists
		_, err = GetProviderUser(db, provider.ID, user.ID)
		assert.ErrorContains(t, err, "not found")

		// but they can be re-created
		_, err = CreateProviderUser(db, provider, user)
		assert.NilError(t, err)
	})
}

func TestProvisionProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("user is created and new identity is linked", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "david@example.com",
				GivenName:  "david",
				FamilyName: "martinez",
				ProviderID: InfraProvider(db).ID,
				Groups:     models.CommaSeparatedStrings{},
				Active:     true,
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			opts := GetIdentityOptions{
				ByName: user.Email,
			}
			identity, err := GetIdentity(db, opts)
			assert.NilError(t, err)
			created, err := GetProviderUser(db, InfraProvider(db).ID, identity.ID)
			assert.NilError(t, err)

			assert.DeepEqual(t, user, created, cmpTimeWithDBPrecision)
		})
		t.Run("user is created and existing identity is linked", func(t *testing.T) {
			identity := &models.Identity{
				Name: "lucy@example.com",
			}
			err := CreateIdentity(db, identity)
			assert.NilError(t, err)

			user := &models.ProviderUser{
				Email:      "lucy@example.com",
				GivenName:  "lucy",
				FamilyName: "",
				ProviderID: InfraProvider(db).ID,
				Groups:     models.CommaSeparatedStrings{},
				Active:     true,
			}

			err = ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			created, err := GetProviderUser(db, InfraProvider(db).ID, identity.ID)
			assert.NilError(t, err)

			assert.DeepEqual(t, user, created, cmpTimeWithDBPrecision)
		})
	})
}

func TestPatchProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("user active status can be patched", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "david@example.com",
				GivenName:  "david",
				FamilyName: "martinez",
				ProviderID: InfraProvider(db).ID,
				Active:     true,
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			user.Active = false

			updated, err := PatchProviderUserActiveStatus(db, user)
			assert.NilError(t, err)
			assert.Equal(t, updated.Active, false)
		})
		t.Run("fields other than active status are ignored in patch", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "lucy@example.com",
				GivenName:  "lucy",
				ProviderID: InfraProvider(db).ID,
				Active:     true,
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			user.Active = false
			user.GivenName = "something else"

			updated, err := PatchProviderUserActiveStatus(db, user)
			assert.NilError(t, err)
			user.GivenName = "Lucy"
			assert.DeepEqual(t, updated, user, cmpTimeWithDBPrecision)
		})
	})
}

func TestUpdateProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("existing user can be updated", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "david@example.com",
				GivenName:  "david",
				FamilyName: "martinez",
				ProviderID: InfraProvider(db).ID,
				Active:     true,
				Groups:     models.CommaSeparatedStrings{},
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			user.Email = "david1@example.com"
			user.GivenName = "Dave"
			user.FamilyName = "Martinez"
			user.Active = false

			err = UpdateProviderUser(db, user)
			assert.NilError(t, err)
			updatedUser, err := GetProviderUser(db, user.ProviderID, user.IdentityID)
			assert.NilError(t, err)
			assert.DeepEqual(t, updatedUser, user, cmpTimeWithDBPrecision)
			opts := GetIdentityOptions{
				ByID: user.IdentityID,
			}
			identity, err := GetIdentity(db, opts)
			assert.NilError(t, err)
			assert.Equal(t, identity.Name, user.Email)
		})
		t.Run("cannot update email of user that exists in multiple providers", func(t *testing.T) {
			provider := &models.Provider{
				Name: "mockta",
				Kind: models.ProviderKindOkta,
			}
			err := CreateProvider(db, provider)
			assert.NilError(t, err)

			existing := &models.ProviderUser{
				Email:      "lucy@example.com",
				GivenName:  "Lucy",
				ProviderID: provider.ID,
				Active:     true,
				Groups:     models.CommaSeparatedStrings{},
			}

			err = ProvisionProviderUser(db, existing)
			assert.NilError(t, err)

			user := &models.ProviderUser{
				Email:      "lucy@example.com",
				GivenName:  "Lucy",
				FamilyName: "",
				ProviderID: InfraProvider(db).ID,
				Active:     true,
				Groups:     models.CommaSeparatedStrings{},
			}

			err = ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			user.Email = "lucy1@example.com"

			err = UpdateProviderUser(db, user)
			assert.ErrorContains(t, err, "cannot update user's email")
		})
	})
}

func TestGetProviderUser(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("existing user is retrieved", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "david@example.com",
				GivenName:  "david",
				FamilyName: "martinez",
				ProviderID: InfraProvider(db).ID,
				Groups:     models.CommaSeparatedStrings{},
				Active:     true,
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)
			opts := GetIdentityOptions{
				ByName: user.Email,
			}
			identity, err := GetIdentity(db, opts)
			assert.NilError(t, err)
			created, err := GetProviderUser(db, InfraProvider(db).ID, identity.ID)
			assert.NilError(t, err)

			assert.DeepEqual(t, user, created, cmpTimeWithDBPrecision)
		})
		t.Run("non-existent identity ID returns error", func(t *testing.T) {
			_, err := GetProviderUser(db, InfraProvider(db).ID, 1234)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("non-existent provider ID returns error", func(t *testing.T) {
			user := &models.ProviderUser{
				Email:      "lucy@example.com",
				GivenName:  "lucy",
				FamilyName: "",
				ProviderID: InfraProvider(db).ID,
				Groups:     models.CommaSeparatedStrings{},
				Active:     true,
			}

			err := ProvisionProviderUser(db, user)
			assert.NilError(t, err)

			_, err = GetProviderUser(db, 1234, user.IdentityID)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestListProviderUsers(t *testing.T) {
	type testCase struct {
		name  string
		setup func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int)
	}

	testCases := []testCase{
		{
			name: "list all provider users",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu := createTestProviderUser(t, tx, provider, "david@example.com")
				return provider.ID, nil, []models.ProviderUser{pu}, 0
			},
		},
		{
			name: "list all provider users invalid provider ID",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				_ = createTestProviderUser(t, tx, provider, "david@example.com")
				return 1234, nil, nil, 0
			},
		},
		{
			name: "limit less than total",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu := createTestProviderUser(t, tx, provider, "david@example.com")
				_ = createTestProviderUser(t, tx, provider, "lucy@example.com")
				return provider.ID, &SCIMParameters{Count: 1}, []models.ProviderUser{pu}, 2
			},
		},
		{
			name: "offset from start",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu1 := createTestProviderUser(t, tx, provider, "david@example.com")
				pu2 := createTestProviderUser(t, tx, provider, "lucy@example.com")
				return provider.ID, &SCIMParameters{StartIndex: 1}, []models.ProviderUser{pu1, pu2}, 2
			},
		},
		{
			name: "equality filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				_ = createTestProviderUser(t, tx, provider, "david@example.com")
				pu := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf(`id eq "%d"`, pu.IdentityID)))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu}, 1
			},
		},
		{
			name: "present filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu1 := createTestProviderUser(t, tx, provider, "david@example.com")
				pu2 := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte("email pr"))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu1, pu2}, 2
			},
		},
		{
			name: "not equal filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu1 := createTestProviderUser(t, tx, provider, "david@example.com")
				pu2 := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("id ne \"%d\"", pu1.IdentityID)))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu2}, 1
			},
		},
		{
			name: "starts with filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				_ = createTestProviderUser(t, tx, provider, "david@example.com")
				pu := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("email sw \"%s\"", "lucy")))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu}, 1
			},
		},
		{
			name: "contains filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				_ = createTestProviderUser(t, tx, provider, "david@example.com")
				pu := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("email co \"%s\"", "lucy")))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu}, 1
			},
		},
		{
			name: "ends with filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu1 := createTestProviderUser(t, tx, provider, "david@example.com")
				pu2 := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("email ew \"%s\"", "example.com")))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu1, pu2}, 2
			},
		},
		{
			name: "logical and filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				_ = createTestProviderUser(t, tx, provider, "david@example.com")
				pu := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("email ew \"%s\" and id eq \"%d\"", "example.com", pu.IdentityID)))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu}, 1
			},
		},
		{
			name: "logical or filter",
			setup: func(t *testing.T, tx *Transaction) (providerID uid.ID, p *SCIMParameters, expected []models.ProviderUser, totalCount int) {
				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOkta,
				}

				err := CreateProvider(tx, provider)
				assert.NilError(t, err)

				pu1 := createTestProviderUser(t, tx, provider, "david@example.com")
				pu2 := createTestProviderUser(t, tx, provider, "lucy@example.com")
				filter, err := filter.ParseFilter([]byte(fmt.Sprintf("email co \"%s\" or email co \"%s\"", "david", "lucy")))
				assert.NilError(t, err)

				return provider.ID, &SCIMParameters{Filter: filter}, []models.ProviderUser{pu1, pu2}, 2
			},
		},
	}

	runDBTests(t, func(t *testing.T, db *DB) {
		org := &models.Organization{Name: "something", Domain: "example.com"}
		assert.NilError(t, CreateOrganization(db, org))

		// create some dummy data for another org to test multi-tenancy
		stmt := `
					INSERT INTO provider_users(identity_id, provider_id, email)
					VALUES (?, ?, ?);
				`
		_, err := db.Exec(stmt, 123, 123, "otherorg@example.com")
		assert.NilError(t, err)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tx := txnForTestCase(t, db, org.ID)

				providerID, p, expected, totalCount := tc.setup(t, tx)

				result, err := ListProviderUsers(tx, ListProviderUsersOptions{ByProviderID: providerID, SCIMParameters: p})

				assert.NilError(t, err)
				assert.DeepEqual(t, result, expected, cmpTimeWithDBPrecision)
				if p != nil {
					assert.Equal(t, p.TotalCount, totalCount, tc.name)
				}
			})
		}
	})
}

func createTestProviderUser(t *testing.T, tx *Transaction, provider *models.Provider, userName string) models.ProviderUser {
	user := &models.Identity{
		Name: userName,
	}
	err := CreateIdentity(tx, user)
	assert.NilError(t, err)

	pu, err := CreateProviderUser(tx, provider, user)
	assert.NilError(t, err)

	pu.Groups = models.CommaSeparatedStrings{}

	return *pu
}
