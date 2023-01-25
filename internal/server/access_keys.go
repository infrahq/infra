package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

func (a *API) ListAccessKeys(c *gin.Context, r *api.ListAccessKeysRequest) (*api.ListResponse[api.AccessKey], error) {
	rCtx := getRequestContext(c)
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
func (a *API) DeleteAccessKey(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteAccessKey(getRequestContext(c), r.ID, "")
}

// DeleteAccessKeys deletes 0 or more access keys by any attribute
func (a *API) DeleteAccessKeys(c *gin.Context, r *api.DeleteAccessKeyRequest) (*api.EmptyResponse, error) {
	return nil, access.DeleteAccessKey(getRequestContext(c), 0, r.Name)
}

func (a *API) CreateAccessKey(c *gin.Context, r *api.CreateAccessKeyRequest) (*api.CreateAccessKeyResponse, error) {
	rCtx := getRequestContext(c)
	accessKey := &models.AccessKey{
		IssuedFor:           r.UserID,
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
		IssuedFor:         accessKey.IssuedFor,
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
			IssuedFor:         latest.IssuedFor,
			ProviderID:        latest.ProviderID,
			Expires:           latest.Expires,
			ExtensionDeadline: latest.InactivityTimeout,
		}
	}

	addVersionHandler(a, http.MethodGet, "/api/access-keys", "0.16.1",
		route[listAccessKeysRequestV0_16_1, *api.ListResponse[accessKeyV0_18_0]]{
			routeSettings: defaultRouteSettingsGet,
			handler: func(c *gin.Context, reqOld *listAccessKeysRequestV0_16_1) (*api.ListResponse[accessKeyV0_18_0], error) {
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
			handler: func(c *gin.Context, reqOld *createAccessKeysRequestV0_18_0) (*createAccessKeyResponseV0_18_0, error) {
				req := &api.CreateAccessKeyRequest{
					UserID:            reqOld.UserID,
					Name:              reqOld.Name,
					Expiry:            reqOld.TTL,
					InactivityTimeout: reqOld.ExtensionDeadline,
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
					IssuedFor:         resp.IssuedFor,
					ProviderID:        resp.ProviderID,
					Expires:           resp.Expires,
					ExtensionDeadline: resp.InactivityTimeout,
					AccessKey:         resp.AccessKey,
				}, nil
			},
		})

	addVersionHandler(a, http.MethodGet, "/api/access-keys", "0.18.0",
		route[api.ListAccessKeysRequest, *api.ListResponse[accessKeyV0_18_0]]{
			handler: func(c *gin.Context, req *api.ListAccessKeysRequest) (*api.ListResponse[accessKeyV0_18_0], error) {
				resp, err := a.ListAccessKeys(c, req)
				return api.CopyListResponse(resp, newAccessKeyV0_18_0FromLatest), err
			},
		})
}
