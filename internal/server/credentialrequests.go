package server

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
)

func (a *API) CreateCredentialRequest(c *gin.Context, r *api.CreateCredentialRequest) (*api.CredentialRequest, error) {
	req, err := access.CreateCredentialRequest(c, r.Destination)
	if err != nil {
		return nil, err
	}
	logging.Debugf("Created Credential Request %+v", req)

	result := req.ToAPI()
	return &result, nil
}

// ListCredentialRequests is a long-polling endpoint that returns Credential Requests awaiting to be filled.
func (a *API) ListCredentialRequests(c *gin.Context, r *api.ListCredentialRequest) (*api.ListCredentialRequestResponse, error) {
	resp, err := access.ListCredentialRequests(c, r.Destination, r.LastUpdateIndex)
	if err != nil {
		return nil, err
	}

	apiResp := &api.ListCredentialRequestResponse{
		Items:          make([]api.CredentialRequest, len(resp.Items)),
		MaxUpdateIndex: resp.MaxUpdateIndex,
	}
	for i := range resp.Items {
		apiResp.Items[i] = resp.Items[i].ToAPI()
	}

	return apiResp, nil
}

// UpdateCredentialRequest is called by the connector to populate authn credentials
func (a *API) UpdateCredentialRequest(c *gin.Context, r *api.UpdateCredentialRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)

	if err := access.UpdateCredentialRequest(rCtx, r); err != nil {
		return nil, err
	}

	return nil, nil
}
