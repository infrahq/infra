package providers

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

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

var azureErrorResponse = `
{
	"error": {
		"code": "InvalidAuthenticationToken",
		"message": "Access token has expired or is not yet valid.",
		"innerError": {
			"date": "2022-11-29T15:37:34",
			"request-id": "aaa",
			"client-request-id": "bbb"
		}
	}
}
`

// this access variable are used to infer what the groups response should be
var validAccess = "valid"

func patchGraphGroupMemberEndpoint(t *testing.T, url string) {
	orig := graphGroupMemberEndpoint
	graphGroupMemberEndpoint = url
	t.Cleanup(func() {
		graphGroupMemberEndpoint = orig
	})
}

func azureHandlers(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/v1.0/me/memberOf", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		// use the access token to infer what the response should be
		if strings.Contains(req.Header.Get("Authorization"), validAccess) {
			w.WriteHeader(http.StatusOK)
			_, err := io.WriteString(w, azureGroupResponse)
			assert.Check(t, err, "failed to write memberOf response")
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := io.WriteString(w, azureErrorResponse)
			assert.Check(t, err, "failed to write memberOf error response")
		}
	})
}

func TestAzure_GetUserInfo(t *testing.T) {
	tests := []struct {
		name         string
		infoResponse string
		access       string
		verifyFunc   func(t *testing.T, info *UserInfoClaims, err error)
	}{
		{
			name:   "error response causes group sync to fail",
			access: "aaa",
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"sub": "o_aaabbbccc",
				"name": "Jim Hopper",
				"family_name": "Hopper",
				"given_name": "Jim",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, info *UserInfoClaims, err error) {
				assert.NilError(t, err)

				expected := UserInfoClaims{
					Name:   "Jim Hopper",
					Groups: []string{},
				}
				assert.DeepEqual(t, *info, expected)
			},
		},
		{
			name:   "deleted user's userinfo response causes sync to fail",
			access: "aaa",
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, info *UserInfoClaims, err error) {
				assert.ErrorContains(t, err, "must include either a name or email")
				assert.Assert(t, info == nil)
			},
		},
		{
			name:   "groups are set from graph response",
			access: validAccess,
			infoResponse: `{
				"sub": "o_aaabbbccc",
				"sub": "o_aaabbbccc",
				"name": "Jim Hopper",
				"family_name": "Hopper",
				"given_name": "Jim",
				"picture": "https://graph.microsoft.com/v1.0/me/photo/$value"
			}`,
			verifyFunc: func(t *testing.T, info *UserInfoClaims, err error) {
				assert.NilError(t, err)

				expected := UserInfoClaims{
					Name:   "Jim Hopper",
					Groups: []string{"Everyone", "Developers"},
				}
				assert.DeepEqual(t, *info, expected)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, ctx := setupOIDCTest(t, test.infoResponse)
			serverURL := server.run(t, azureHandlers)
			provider := NewOIDCClient(models.Provider{Kind: models.ProviderKindAzure, URL: serverURL, ClientID: "invalid"}, "invalid", "https://example.com/callback")
			patchGraphGroupMemberEndpoint(t, "https://"+serverURL+"/v1.0/me/memberOf")
			info, err := provider.GetUserInfo(ctx, &models.ProviderUser{AccessToken: models.EncryptedAtRest(test.access), RefreshToken: "bbb", ExpiresAt: time.Now().UTC().Add(5 * time.Minute)})
			test.verifyFunc(t, info, err)
		})
	}
}
