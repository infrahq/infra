package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

var googleWorkspaceGroupResponse = `{
    "kind": "directory#groups",
    "groups": [
     {
      "kind": "directory#group",
      "id": "group's unique ID",
      "etag": "group's unique ETag",
      "email": "sales_group@example.com",
      "name": "sale group",
      "directMembersCount": "5",
      "description": "Sales group"
     },
     {
      "kind": "directory#group",
      "id": "group's unique ID",
      "etag": "group's unique ETag",
      "email": "support_group.com",
      "name": "support group",
      "directMembersCount": "5",
      "description": "Support group"
     }
  ],
   "nextPakeToken": "NNNNN"
}`

func googleHandlers(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/o/oauth2/token", func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("apple")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err := io.WriteString(w, `
		{
			"access_token": "1/fFAGRNJru1FTz70BzhT3Zg",
			"expires_in": 3920,
			"token_type": "Bearer",
			"scope": "https://www.googleapis.com/auth/drive.metadata.readonly",
			"refresh_token": "1//xEoDL4iW3cxlI7yDbSRFYNG01kVKM2C-259HOF2aQbI"
		  }
		`)
		assert.Check(t, err, "failed to write token response")
	})
	mux.HandleFunc("/admin/directory/v1/groups?alt=json&prettyPrint=false&userKey=", func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("banana")
		w.Header().Add("Content-Type", "application/json")
		_, err := io.WriteString(w, googleWorkspaceGroupResponse)
		w.WriteHeader(200)
		assert.Check(t, err, "failed to write memberOf response")
	})
}

func TestGoogle_GetUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		infoResponse   string
		groupsResponse []string
		verifyFunc     func(t *testing.T, info *UserInfoClaims, err error)
	}{
		{
			name: "invalid credentials cause sync to fail",
			infoResponse: `{
				"error": "invalid_request",
				"error_description": "Invalid Credentials"
			}`,
			groupsResponse: []string{},
			verifyFunc: func(t *testing.T, info *UserInfoClaims, err error) {
				assert.ErrorContains(t, err, "could not get user info from provider: claim must include either a name or email")
				assert.Assert(t, info == nil)
			},
		},
		{
			name: "groups are set from Google API response",
			infoResponse: `{
				"sub": "1",
				"picture": "https://lh3.googleusercontent.com",
				"email": "hello@example.com",
				"email_verified": true,
				"hd": "example.com"
			}`,
			groupsResponse: []string{"apples", "oranges"},
			verifyFunc: func(t *testing.T, info *UserInfoClaims, err error) {
				assert.NilError(t, err)
				assert.Equal(t, info.Email, "hello@example.com")
				assert.Assert(t, reflect.DeepEqual(info.Groups, []string{"apples", "oranges"}))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, ctx := setupOIDCTest(t, test.infoResponse)
			serverURL := server.run(t, googleHandlers)
			provider := models.Provider{
				Kind:             models.ProviderKindGoogle,
				URL:              serverURL,
				ClientID:         "invalid",
				PrivateKey:       models.EncryptedAtRest("-----BEGIN PRIVATE KEY-----\naaa=\n-----END PRIVATE KEY-----\n"),
				ClientEmail:      "something",
				DomainAdminEmail: "admin",
			}
			oidcClient := NewOIDCClient(provider, "invalid", "http://localhost:8301")
			info, err := oidcClient.GetUserInfo(context.WithValue(ctx, testGroupsKey{}, test.groupsResponse), &models.ProviderUser{AccessToken: "aaa", RefreshToken: "bbb", ExpiresAt: time.Now().UTC().Add(5 * time.Minute)})
			test.verifyFunc(t, info, err)
		})
	}
}
