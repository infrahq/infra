package providers

import (
	"io"
	"net/http"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2/jwt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

const azureGroupResponse = `{
	"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#directoryObjects",
	"value": [
		{
			"@odata.type": "#microsoft.graph.directoryRole",
			"id": "aaa",
			"deletedDateTime": null,
			"description": "Can manage all aspects of Azure AD and Microsoft services that use Azure AD identities.",
			"displayName": "Global Administrator",
			"roleTemplateId": "bbb"
		},
		{
			"@odata.type": "#microsoft.graph.group",
			"id": "ccc",
			"deletedDateTime": null,
			"classification": null,
			"createdDateTime": "2022-06-07T20:40:20Z",
			"creationOptions": [],
			"description": null,
			"displayName": "Everyone",
			"expirationDateTime": null,
			"groupTypes": [],
			"isAssignableToRole": null,
			"mail": null,
			"mailEnabled": false,
			"mailNickname": "eee",
			"membershipRule": null,
			"membershipRuleProcessingState": null,
			"onPremisesDomainName": null,
			"onPremisesLastSyncDateTime": null,
			"onPremisesNetBiosName": null,
			"onPremisesSamAccountName": null,
			"onPremisesSecurityIdentifier": null,
			"onPremisesSyncEnabled": null,
			"preferredDataLocation": null,
			"preferredLanguage": null,
			"proxyAddresses": [],
			"renewedDateTime": "2022-06-07T20:40:20Z",
			"resourceBehaviorOptions": [],
			"resourceProvisioningOptions": [],
			"securityEnabled": true,
			"securityIdentifier": "qqq",
			"theme": null,
			"visibility": null,
			"onPremisesProvisioningErrors": []
		},
		{
			"@odata.type": "#microsoft.graph.group",
			"id": "ccc",
			"deletedDateTime": null,
			"classification": null,
			"createdDateTime": "2022-06-07T20:40:20Z",
			"creationOptions": [],
			"description": null,
			"displayName": "Developers",
			"expirationDateTime": null,
			"groupTypes": [],
			"isAssignableToRole": null,
			"mail": null,
			"mailEnabled": false,
			"mailNickname": "eee",
			"membershipRule": null,
			"membershipRuleProcessingState": null,
			"onPremisesDomainName": null,
			"onPremisesLastSyncDateTime": null,
			"onPremisesNetBiosName": null,
			"onPremisesSamAccountName": null,
			"onPremisesSecurityIdentifier": null,
			"onPremisesSyncEnabled": null,
			"preferredDataLocation": null,
			"preferredLanguage": null,
			"proxyAddresses": [],
			"renewedDateTime": "2022-06-07T20:40:20Z",
			"resourceBehaviorOptions": [],
			"resourceProvisioningOptions": [],
			"securityEnabled": true,
			"securityIdentifier": "qqq",
			"theme": null,
			"visibility": null,
			"onPremisesProvisioningErrors": []
		}
	]
}`

func patchGraphGroupMemberEndpoint(t *testing.T, url string) {
	orig := graphGroupMemberEndpoint
	graphGroupMemberEndpoint = url
	t.Cleanup(func() {
		graphGroupMemberEndpoint = orig
	})
}

func azureHandlers(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/v1.0/me/memberOf/fail", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(403)
	})
	mux.HandleFunc("/v1.0/me/memberOf", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_, err := io.WriteString(w, azureGroupResponse)
		w.WriteHeader(200)
		if err != nil {
			assert.Check(t, err, "failed to write memberOf response")
		}
	})
}

func TestSyncAzureProviderUser(t *testing.T) {
	db := setupDB(t)

	provider := &models.Provider{
		Name: "mock-azure",
		Kind: models.AzureKind,
	}

	err := data.CreateProvider(db, provider)
	assert.NilError(t, err)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, serverURL string) *models.Identity
		infoResponse string
		verifyFunc   func(t *testing.T, err error, user *models.Identity)
	}{
		{
			name: "invalid/expired access token is updated",
			setupFunc: func(t *testing.T, serverURL string) *models.Identity {
				user := &models.Identity{
					Name: "sharrington@example.com",
				}

				err = data.CreateIdentity(db, user)
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

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"name": "Steve Harrington",
				"family_name": "Harrington",
				"given_name": "Steve",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.NilError(t, err)

				pu, err := data.GetProviderUser(db, provider.ID, user.ID)
				assert.NilError(t, err)

				assert.Assert(t, pu.AccessToken != "aaa")
				assert.Equal(t, string(pu.RefreshToken), "bbb")
				assert.Assert(t, cmpAPITimeWithThreshold(pu.ExpiresAt, time.Now().UTC().Add(1*time.Hour)))
				assert.Assert(t, cmpAPITimeWithThreshold(pu.LastUpdate, time.Now().UTC()))
			},
		},
		{
			name: "deleted user's userinfo response causes sync to fail",
			setupFunc: func(t *testing.T, serverURL string) *models.Identity {
				user := &models.Identity{
					Name: "rbuckleyn@example.com",
				}

				err = data.CreateIdentity(db, user)
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

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.ErrorContains(t, err, "could not get user info from provider")
			},
		},
		{
			name: "failure to sync groups does not fail sync",
			setupFunc: func(t *testing.T, serverURL string) *models.Identity {
				patchGraphGroupMemberEndpoint(t, "https://"+serverURL+"/v1.0/me/memberOf/fail")

				user := &models.Identity{
					Name: "nwheeler@example.com",
				}

				err = data.CreateIdentity(db, user)
				assert.NilError(t, err)

				pu := &models.ProviderUser{
					ProviderID: provider.ID,
					IdentityID: user.ID,

					Email:        user.Name,
					RedirectURL:  "http://example.com",
					AccessToken:  models.EncryptedAtRest("aaa"), // this is used to fail the groups call, in reality this token should be valid
					RefreshToken: models.EncryptedAtRest("bbb"),
					ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
					LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
				}

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"sub": "o_aaabbbccc",
				"name": "Nancy Wheeler",
				"family_name": "Wheeler",
				"given_name": "Nancy",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.NilError(t, err)
				assert.Assert(t, len(user.Groups) == 0)
			},
		},
		{
			name: "groups are set from graph response",
			setupFunc: func(t *testing.T, serverURL string) *models.Identity {
				patchGraphGroupMemberEndpoint(t, "https://"+serverURL+"/v1.0/me/memberOf")

				user := &models.Identity{
					Name: "jhopper@example.com",
				}

				err = data.CreateIdentity(db, user)
				assert.NilError(t, err)

				pu := &models.ProviderUser{
					ProviderID: provider.ID,
					IdentityID: user.ID,

					Email:        user.Name,
					RedirectURL:  "http://example.com",
					AccessToken:  models.EncryptedAtRest("aaa"), // this is used to fail the groups call, in reality this token should be valid
					RefreshToken: models.EncryptedAtRest("bbb"),
					ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
					LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
				}

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"sub": "o_aaabbbccc",
				"name": "Jim Hopper",
				"family_name": "Hopper",
				"given_name": "Jim",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.NilError(t, err)

				pu, err := data.GetProviderUser(db, provider.ID, user.ID)
				assert.NilError(t, err)
				assert.Assert(t, cmpAPITimeWithThreshold(pu.LastUpdate, time.Now().UTC()))

				assert.Assert(t, len(pu.Groups) == 2)

				groups := make(map[string]bool)
				for _, g := range pu.Groups {
					groups[g] = true
				}

				assert.Assert(t, groups["Everyone"])
				assert.Assert(t, groups["Developers"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, ctx := setupOIDCTest(t, test.infoResponse)
			serverURL := server.run(t, azureHandlers)
			oidc := &oidcImplementation{
				ProviderID:   provider.ID,
				Domain:       serverURL,
				ClientID:     "whatever",
				ClientSecret: "whatever",
				RedirectURL:  "http://localhost:8301",
			}
			azure := &azure{OIDC: oidc}

			now := time.Now().UTC()

			claims := jwt.Claims{
				Audience:  jwt.Audience([]string{"whatever"}),
				NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
				Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    serverURL,
			}

			body, err := testTokenResponse(claims, server.signingKey, "hello@example.com")
			assert.NilError(t, err)

			server.tokenResponse = tokenResponse{
				code: 200,
				body: body,
			}

			user := test.setupFunc(t, serverURL)
			err = azure.SyncProviderUser(ctx, db, user, provider)
			test.verifyFunc(t, err, user)
		})
	}
}
