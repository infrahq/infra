package server

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func (a *API) addRequestRewrites() {
	// all request migrations go here
	type oldListAccessKeysRequest struct {
		UserID      uid.ID `form:"user_id"`
		Name        string `form:"name"`
		ShowExpired bool   `form:"show_expired"`
		api.PaginationRequest
	}
	type newListAccessKeysRequest struct {
		UserID      uid.ID `form:"userID"`
		Name        string `form:"name"`
		ShowExpired bool   `form:"showExpired"`
		api.PaginationRequest
	}
	addRequestRewrite(a, http.MethodGet, "/api/access-keys", "0.16.1", func(o oldListAccessKeysRequest) newListAccessKeysRequest {
		return newListAccessKeysRequest(o)
	})
	type createAccessKeysRequestV0_18_0 struct {
		UserID            uid.ID       `json:"userID"`
		Name              string       `json:"name"`
		TTL               api.Duration `json:"ttl"`
		ExtensionDeadline api.Duration `json:"extensionDeadline"`
	}
	addRequestRewrite(a, http.MethodPost, "/api/access-keys", "0.18.0", func(o createAccessKeysRequestV0_18_0) api.CreateAccessKeyRequest {
		return api.CreateAccessKeyRequest{
			UserID:            o.UserID,
			Name:              o.Name,
			Expiry:            o.TTL,
			InactivityTimeout: o.ExtensionDeadline,
		}
	})
	type signupOrgV0_19_0 struct {
		Name      string `json:"name"`
		Subdomain string `json:"subDomain"`
	}
	type signupRequestV0_19_0 struct {
		Name     string           `json:"name"`
		Password string           `json:"password"`
		Org      signupOrgV0_19_0 `json:"org"`
	}
	addRequestRewrite(a, http.MethodPost, "/api/signup", "0.19.0", func(oldRequest signupRequestV0_19_0) api.SignupRequest {
		return api.SignupRequest{
			User: &api.SignupUser{
				UserName: oldRequest.Name,
				Password: oldRequest.Password,
			},
			OrgName:   oldRequest.Org.Name,
			Subdomain: oldRequest.Org.Subdomain,
		}
	})
}

func (a *API) addResponseRewrites() {
	type accessKeyV0_18_0 struct {
		ID                uid.ID   `json:"id"`
		Created           api.Time `json:"created"`
		LastUsed          api.Time `json:"lastUsed"`
		Name              string   `json:"name"`
		IssuedForName     string   `json:"issuedForName"`
		IssuedFor         uid.ID   `json:"issuedFor"`
		ProviderID        uid.ID   `json:"providerID"`
		Expires           api.Time `json:"expires"`
		ExtensionDeadline api.Time `json:"extensionDeadline"`
	}
	addResponseRewrite(a, http.MethodGet, "/api/access-keys", "0.18.0", func(newResponse *api.ListResponse[api.AccessKey]) *api.ListResponse[accessKeyV0_18_0] {
		return api.NewListResponse(newResponse.Items, newResponse.PaginationResponse, func(newResponseItem api.AccessKey) accessKeyV0_18_0 {
			return accessKeyV0_18_0{
				ID:                newResponseItem.ID,
				Created:           newResponseItem.Created,
				LastUsed:          newResponseItem.LastUsed,
				Name:              newResponseItem.Name,
				IssuedForName:     newResponseItem.IssuedForName,
				IssuedFor:         newResponseItem.IssuedFor,
				ProviderID:        newResponseItem.ProviderID,
				Expires:           newResponseItem.Expires,
				ExtensionDeadline: newResponseItem.InactivityTimeout,
			}
		})
	})
	type createAccessKeyV0_18_0 struct {
		ID                uid.ID   `json:"id"`
		Created           api.Time `json:"created"`
		Name              string   `json:"name"`
		IssuedFor         uid.ID   `json:"issuedFor"`
		ProviderID        uid.ID   `json:"providerID"`
		Expires           api.Time `json:"expires"`
		ExtensionDeadline api.Time `json:"extensionDeadline"`
		AccessKey         string   `json:"accessKey"`
	}
	addResponseRewrite(a, http.MethodPost, "/api/access-keys", "0.18.0", func(newResponse *api.CreateAccessKeyResponse) *createAccessKeyV0_18_0 {
		return &createAccessKeyV0_18_0{
			ID:                newResponse.ID,
			Created:           newResponse.Created,
			Name:              newResponse.Name,
			IssuedFor:         newResponse.IssuedFor,
			ProviderID:        newResponse.ProviderID,
			Expires:           newResponse.Expires,
			ExtensionDeadline: newResponse.InactivityTimeout,
			AccessKey:         newResponse.AccessKey,
		}
	})

	type grantsV0_18_1 struct {
		ID        uid.ID   `json:"id"`
		Created   api.Time `json:"created"`
		CreatedBy uid.ID   `json:"created_by"`
		Updated   api.Time `json:"updated"`
		User      uid.ID   `json:"user,omitempty"`
		Group     uid.ID   `json:"group,omitempty"`
		Privilege string   `json:"privilege"`
		Resource  string   `json:"resource"`
	}
	addResponseRewrite(a, http.MethodGet, "/api/grants", "0.18.1", func(newResponse *api.ListResponse[api.Grant]) *api.ListResponse[grantsV0_18_1] {
		return api.NewListResponse(newResponse.Items, newResponse.PaginationResponse, func(newResponseItem api.Grant) grantsV0_18_1 {
			return grantsV0_18_1{
				ID:        newResponseItem.ID,
				Created:   newResponseItem.Created,
				CreatedBy: newResponseItem.CreatedBy,
				Updated:   newResponseItem.Updated,
				User:      newResponseItem.User,
				Group:     newResponseItem.Group,
				Privilege: newResponseItem.Privilege,
				Resource:  newResponseItem.Resource,
			}
		})
	})
	addResponseRewrite(a, http.MethodGet, "/api/grants/:id", "0.18.1", func(newResponse *api.Grant) *grantsV0_18_1 {
		return &grantsV0_18_1{
			ID:        newResponse.ID,
			Created:   newResponse.Created,
			CreatedBy: newResponse.CreatedBy,
			Updated:   newResponse.Updated,
			User:      newResponse.User,
			Group:     newResponse.Group,
			Privilege: newResponse.Privilege,
			Resource:  newResponse.Resource,
		}
	})
	type createGrantsV0_18_1 struct {
		*grantsV0_18_1 `json:",inline"`
		WasCreated     bool `json:"wasCreated"`
	}
	addResponseRewrite(a, http.MethodPost, "/api/grants", "0.18.1", func(newResponse *api.CreateGrantResponse) *createGrantsV0_18_1 {
		return &createGrantsV0_18_1{
			grantsV0_18_1: &grantsV0_18_1{
				ID:        newResponse.ID,
				Created:   newResponse.Created,
				CreatedBy: newResponse.CreatedBy,
				Updated:   newResponse.Updated,
				User:      newResponse.User,
				Group:     newResponse.Group,
				Privilege: newResponse.Privilege,
				Resource:  newResponse.Resource,
			},
			WasCreated: newResponse.WasCreated,
		}
	})
	// all response migrations go here
}

func (a *API) addRewrites() {
	a.addRequestRewrites()
	a.addResponseRewrites()
}

// addRedirects for API endpoints that have moved to a different path
func (a *API) addRedirects() {
}

func (a *API) deprecatedRoutes(noAuthnNoOrg *routeGroup) {
	// CLI clients before v0.14.4 rely on sign-up being false to continue with login
	type SignupEnabledResponse struct {
		Enabled bool `json:"enabled"`
	}

	add(a, noAuthnNoOrg, http.MethodGet, "/api/signup", route[api.EmptyRequest, *SignupEnabledResponse]{
		handler: func(c *gin.Context, _ *api.EmptyRequest) (*SignupEnabledResponse, error) {
			return &SignupEnabledResponse{Enabled: false}, nil
		},
		routeSettings: routeSettings{
			omitFromTelemetry: true,
			omitFromDocs:      true,
			txnOptions:        &sql.TxOptions{ReadOnly: true},
		},
	})
}
