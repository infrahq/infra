package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
)

var createDestinationCredentialRoute = route[api.CreateDestinationCredentialRequest, *api.DestinationCredential]{
	handler: CreateDestinationCredential,
	routeSettings: routeSettings{
		transactionCommitsEarly: true,
	},
}

func CreateDestinationCredential(c *gin.Context, r *api.CreateDestinationCredentialRequest) (*api.DestinationCredential, error) {
	req, err := access.CreateDestinationCredential(c, r.Destination)
	if err != nil {
		return nil, err
	}
	logging.Debugf("Created destination credential %+v", req)

	result := req.ToAPI()
	return &result, nil
}

var listDestinationCredentialRoute = route[api.ListDestinationCredentialRequest, *api.ListDestinationCredentialResponse]{
	handler: ListDestinationCredentials,
	routeSettings: routeSettings{
		transactionCommitsEarly: true,
	},
}

// ListDestinationCredentials is a long-polling endpoint that returns destination credentials awaiting to be filled.
func ListDestinationCredentials(c *gin.Context, r *api.ListDestinationCredentialRequest) (*api.ListDestinationCredentialResponse, error) {
	resp, err := access.ListDestinationCredentials(c, r.Destination, r.LastUpdateIndex)
	if err != nil {
		return nil, err
	}

	apiResp := &api.ListDestinationCredentialResponse{
		Items:          make([]api.DestinationCredential, len(resp.Items)),
		MaxUpdateIndex: resp.MaxUpdateIndex,
	}
	for i := range resp.Items {
		apiResp.Items[i] = resp.Items[i].ToAPI()
	}

	return apiResp, nil
}

// AnswerDestinationCredential is called by the connector to populate authn credentials
func AnswerDestinationCredential(c *gin.Context, r *api.AnswerDestinationCredentialRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)

	if err := access.AnswerDestinationCredential(rCtx, r); err != nil {
		return nil, err
	}

	return nil, nil
}
