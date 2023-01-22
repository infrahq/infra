package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

func (a *API) ListAccessKeys(rCtx access.RequestContext, r *api.ListAccessKeysRequest) (*api.ListResponse[api.AccessKey], error) {
	
	p := PaginationFromRequest(r.PaginationRequest)
	accessKeys, err := access.ListAccessKeys(rCtx, r.UserID, r.Name, r.ShowExpired, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(accessKeys, PaginationToResponse(p), func(accessKey models.AccessKey) api.AccessKey {
		return *accessKey.ToAPI()
	})

	return result, nil
}

// DeleteAccessKey deletes an access key by id
func (a *API) DeleteAccessKey(rCtx access.RequestContext, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteAccessKey(getRequestContext(c), r.ID, "")
}

// DeleteAccessKeys deletes 0 or more access keys by any attribute
func (a *API) DeleteAccessKeys(rCtx access.RequestContext, r *api.DeleteAccessKeyRequest) (*api.EmptyResponse, error) {
	return nil, access.DeleteAccessKey(getRequestContext(c), 0, r.Name)
}

func (a *API) CreateAccessKey(rCtx access.RequestContext, r *api.CreateAccessKeyRequest) (*api.CreateAccessKeyResponse, error) {
	
	accessKey := &models.AccessKey{
		IssuedForID:         r.IssuedForID,
		IssuedForKind:       models.IssuedKind(r.IssuedForKind),
		Name:                r.Name,
		ExpiresAt:           time.Now().UTC().Add(time.Duration(r.Expiry)),
		InactivityExtension: time.Duration(r.InactivityTimeout),
		InactivityTimeout:   time.Now().UTC().Add(time.Duration(r.InactivityTimeout)),
	}

	raw, err := access.CreateAccessKey(rCtx, accessKey)
	if err != nil {
		return nil, err
	}

	return &api.CreateAccessKeyResponse{
		ID:                accessKey.ID,
		Created:           api.Time(accessKey.CreatedAt),
		Name:              accessKey.Name,
		IssuedForID:       accessKey.IssuedForID,
		IssuedForKind:     accessKey.IssuedForKind.String(),
		Expires:           api.Time(accessKey.ExpiresAt),
		InactivityTimeout: api.Time(accessKey.InactivityTimeout),
		AccessKey:         raw,
	}, nil
}

// See docs/dev/api-versioned-handlers.md for a guide to adding new version handlers.
func (a *API) addPreviousVersionHandlersAccessKey() {
	type listAccessKeysRequestV0_16_1 struct {
		UserID      uid.ID `form:"user_id"`
		Name        string `form:"name"`
		ShowExpired bool   `form:"show_expired"`
		api.PaginationRequest
	}
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
	newAccessKeyV0_18_0FromLatest := func(latest api.AccessKey) accessKeyV0_18_0 {
		return accessKeyV0_18_0{
			ID:                latest.ID,
			Created:           latest.Created,
			LastUsed:          latest.LastUsed,
			Name:              latest.Name,
			IssuedForName:     latest.IssuedForName,
			IssuedFor:         latest.IssuedForID,
			ProviderID:        latest.ProviderID,
			Expires:           latest.Expires,
			ExtensionDeadline: latest.InactivityTimeout,
		}
	}

	addVersionHandler(a, http.MethodGet, "/api/access-keys", "0.16.1",
		route[listAccessKeysRequestV0_16_1, *api.ListResponse[accessKeyV0_18_0]]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(rCtx access.RequestContext, reqOld *listAccessKeysRequestV0_16_1) (*api.ListResponse[accessKeyV0_18_0], error) {
				req := &api.ListAccessKeysRequest{
					UserID:            reqOld.UserID,
					Name:              reqOld.Name,
					ShowExpired:       reqOld.ShowExpired,
					PaginationRequest: reqOld.PaginationRequest,
				}
				if err := validate.Validate(req); err != nil {
					return nil, err
				}
				resp, err := a.ListAccessKeys(c, req)
				return api.CopyListResponse(resp, newAccessKeyV0_18_0FromLatest), err
			},
		})

	type createAccessKeysRequestV0_18_0 struct {
		UserID            uid.ID       `json:"userID"`
		Name              string       `json:"name"`
		TTL               api.Duration `json:"ttl"`
		ExtensionDeadline api.Duration `json:"extensionDeadline"`
	}
	type createAccessKeyResponseV0_18_0 struct {
		ID                uid.ID   `json:"id"`
		Created           api.Time `json:"created"`
		Name              string   `json:"name"`
		IssuedFor         uid.ID   `json:"issuedFor"`
		ProviderID        uid.ID   `json:"providerID"`
		Expires           api.Time `json:"expires"`
		ExtensionDeadline api.Time `json:"extensionDeadline"`
		AccessKey         string   `json:"accessKey"`
	}
	addVersionHandler(a, http.MethodPost, "/api/access-keys", "0.18.0",
		route[createAccessKeysRequestV0_18_0, *createAccessKeyResponseV0_18_0]{
			handler: func(rCtx access.RequestContext, reqOld *createAccessKeysRequestV0_18_0) (*createAccessKeyResponseV0_18_0, error) {
				req := &api.CreateAccessKeyRequest{
					IssuedForID:       reqOld.UserID,
					IssuedForKind:     models.IssuedForKindUser.String(),
					Name:              reqOld.Name,
					Expiry:            reqOld.TTL,
					InactivityTimeout: reqOld.ExtensionDeadline,
				}
				// check if this is an access key being issued for identity provider scim
				if strings.HasSuffix(req.Name, "-scim") {
					req.IssuedForKind = api.KeyIssuedForKindProvider
				}
				if err := validate.Validate(req); err != nil {
					return nil, err
				}
				resp, err := a.CreateAccessKey(c, req)
				if err != nil {
					return nil, err
				}
				return &createAccessKeyResponseV0_18_0{
					ID:                resp.ID,
					Created:           resp.Created,
					Name:              resp.Name,
					IssuedFor:         resp.IssuedForID,
					ProviderID:        resp.ProviderID,
					Expires:           resp.Expires,
					ExtensionDeadline: resp.InactivityTimeout,
					AccessKey:         resp.AccessKey,
				}, nil
			},
		})

	addVersionHandler(a, http.MethodGet, "/api/access-keys", "0.18.0",
		route[api.ListAccessKeysRequest, *api.ListResponse[accessKeyV0_18_0]]{
			handler: func(rCtx access.RequestContext, req *api.ListAccessKeysRequest) (*api.ListResponse[accessKeyV0_18_0], error) {
				resp, err := a.ListAccessKeys(c, req)
				return api.CopyListResponse(resp, newAccessKeyV0_18_0FromLatest), err
			},
		})

	type createAccessKeyRequestV0_20_0 struct {
		UserID            uid.ID       `json:"userID"`
		IssuedForID       uid.ID       `json:"issuedForID"`
		Name              string       `json:"name"`
		Expiry            api.Duration `json:"expiry" note:"maximum time valid"`
		InactivityTimeout api.Duration `json:"inactivityTimeout" note:"key must be used within this duration to remain valid"`
	}
	type createAccessKeyResponseV0_20_0 struct {
		ID                uid.ID   `json:"id"`
		Created           api.Time `json:"created"`
		Name              string   `json:"name"`
		IssuedFor         uid.ID   `json:"issuedFor"`
		ProviderID        uid.ID   `json:"providerID"`
		Expires           api.Time `json:"expires" note:"after this deadline the key is no longer valid"`
		InactivityTimeout api.Time `json:"inactivityTimeout" note:"the key must be used by this time to remain valid"`
		AccessKey         string   `json:"accessKey"`
	}
	addVersionHandler(a,
		http.MethodPost, "/api/access-keys", "0.20.0",
		route[createAccessKeyRequestV0_20_0, *createAccessKeyResponseV0_20_0]{
			handler: func(c *gin.Context, reqOld *createAccessKeyRequestV0_20_0) (*createAccessKeyResponseV0_20_0, error) {
				iss := reqOld.UserID
				if iss == 0 {
					// try setting this from the new field
					iss = reqOld.IssuedForID
				}
				req := &api.CreateAccessKeyRequest{
					IssuedForID:       iss,
					IssuedForKind:     api.KeyIssuedForKindUser,
					Name:              reqOld.Name,
					Expiry:            reqOld.Expiry,
					InactivityTimeout: reqOld.InactivityTimeout,
				}
				// check if this is an access key being issued for identity provider scim
				if strings.HasSuffix(req.Name, "-scim") {
					req.IssuedForKind = api.KeyIssuedForKindProvider
				}

				resp, err := a.CreateAccessKey(c, req)
				if err != nil {
					return nil, err
				}

				return &createAccessKeyResponseV0_20_0{
					ID:                resp.ID,
					Created:           resp.Created,
					Name:              resp.Name,
					IssuedFor:         resp.IssuedForID,
					ProviderID:        resp.ProviderID,
					Expires:           resp.Expires,
					InactivityTimeout: resp.InactivityTimeout,
					AccessKey:         resp.AccessKey,
				}, nil
			},
		})

	type accessKeyV0_20_0 struct {
		ID                uid.ID   `json:"id"`
		Created           api.Time `json:"created"`
		LastUsed          api.Time `json:"lastUsed"`
		Name              string   `json:"name"`
		IssuedForUser     string   `json:"issuedForUser"`
		IssuedFor         uid.ID   `json:"issuedFor"`
		ProviderID        uid.ID   `json:"providerID"`
		Expires           api.Time `json:"expires"`
		InactivityTimeout api.Time `json:"inactivityTimeout"`
		Scopes            []string `json:"scopes"`
	}
	newAccessKeyV0_20_0FromLatest := func(latest api.AccessKey) accessKeyV0_20_0 {
		return accessKeyV0_20_0{
			ID:                latest.ID,
			Created:           latest.Created,
			LastUsed:          latest.LastUsed,
			Name:              latest.Name,
			IssuedForUser:     latest.IssuedForName,
			IssuedFor:         latest.IssuedForID,
			ProviderID:        latest.ProviderID,
			Expires:           latest.Expires,
			InactivityTimeout: latest.InactivityTimeout,
			Scopes:            latest.Scopes,
		}
	}
	addVersionHandler(a,
		http.MethodGet, "/api/access-keys", "0.20.0",
		route[api.ListAccessKeysRequest, *api.ListResponse[accessKeyV0_20_0]]{
			handler: func(c *gin.Context, req *api.ListAccessKeysRequest) (*api.ListResponse[accessKeyV0_20_0], error) {
				resp, err := a.ListAccessKeys(c, req)
				return api.CopyListResponse(resp, newAccessKeyV0_20_0FromLatest), err
			},
		})
}
