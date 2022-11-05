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
	type createProviderRequestV0_16_1 struct {
		Name           string                      `json:"name" example:"okta"`
		Kind           string                      `json:"kind" example:"okta"`
		URL            string                      `json:"url" example:"infrahq.okta.com"`
		ClientID       string                      `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
		ClientSecret   string                      `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
		AllowedDomains []string                    `json:"allowedDomains" example:"['example.com', 'infrahq.com']"`
		API            *api.ProviderAPICredentials `json:"api"`
	}
	addRequestRewrite(a, http.MethodPost, "/api/providers", "0.16.1", func(oldRequest createProviderRequestV0_16_1) api.CreateProviderRequest {
		return api.CreateProviderRequest{
			Name: oldRequest.Name,
			Kind: oldRequest.Kind,
			Client: &api.OIDCClient{
				URL:            oldRequest.URL,
				ClientID:       oldRequest.ClientID,
				ClientSecret:   oldRequest.ClientSecret,
				AllowedDomains: oldRequest.AllowedDomains,
				API:            oldRequest.API,
			},
		}
	})
	type updateProviderRequestV0_16_1 struct {
		ID             uid.ID                      `uri:"id" json:"-"`
		Name           string                      `json:"name" example:"okta"`
		URL            string                      `json:"url" example:"infrahq.okta.com"`
		ClientID       string                      `json:"clientID" example:"0oapn0qwiQPiMIyR35d6"`
		ClientSecret   string                      `json:"clientSecret" example:"jmda5eG93ax3jMDxTGrbHd_TBGT6kgNZtrCugLbU"`
		AllowedDomains []string                    `json:"allowedDomains" example:"['example.com', 'infrahq.com']"`
		Kind           string                      `json:"kind" example:"oidc"`
		API            *api.ProviderAPICredentials `json:"api"`
	}
	addRequestRewrite(a, http.MethodPut, "/api/providers", "0.16.1", func(oldRequest updateProviderRequestV0_16_1) api.UpdateProviderRequest {
		return api.UpdateProviderRequest{
			ID:   oldRequest.ID,
			Name: oldRequest.Name,
			Kind: oldRequest.Kind,
			Client: &api.OIDCClient{
				URL:            oldRequest.URL,
				ClientID:       oldRequest.ClientID,
				ClientSecret:   oldRequest.ClientSecret,
				AllowedDomains: oldRequest.AllowedDomains,
				API:            oldRequest.API,
			},
		}
	})
}

func (a *API) addResponseRewrites() {
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
