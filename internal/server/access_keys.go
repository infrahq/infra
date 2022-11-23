package server

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListAccessKeys(c *gin.Context, r *api.ListAccessKeysRequest) (*api.ListResponse[api.AccessKey], error) {
	p := PaginationFromRequest(r.PaginationRequest)
	accessKeys, err := access.ListAccessKeys(c, r.UserID, r.Name, r.ShowExpired, &p)
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
	accessKey := &models.AccessKey{
		IssuedFor:           r.UserID,
		Name:                r.Name,
		ExpiresAt:           time.Now().UTC().Add(time.Duration(r.Expiry)),
		InactivityExtension: time.Duration(r.InactivityTimeout),
		InactivityTimeout:   time.Now().UTC().Add(time.Duration(r.InactivityTimeout)),
	}

	raw, err := access.CreateAccessKey(c, accessKey)
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
