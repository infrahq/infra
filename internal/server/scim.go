package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
)

var listProviderUsersRoute = route[api.SCIMParametersRequest, *api.ListProviderUsersResponse]{
	handler: ListProviderUsers,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

func ListProviderUsers(c *gin.Context, r *api.SCIMParametersRequest) (*api.ListProviderUsersResponse, error) {
	p := data.SCIMParameters{
		StartIndex: r.StartIndex,
		Count:      r.Count,
	}
	providerID := getRequestContext(c).Authenticated.AccessKey.IssuedFor // TODO: ability to create tokens for providers
	if providerID == 0 {
		// should not happen
		return nil, fmt.Errorf("unable to list provider users, access key does not specify 'issued for'")
	}

	users, err := access.ListProviderUsers(c, providerID, &p)
	if err != nil {
		return nil, err
	}

	result := &api.ListProviderUsersResponse{
		Schemas:      []string{api.ListResponseSchema},
		TotalResults: p.TotalCount,
		StartIndex:   p.StartIndex,
		ItemsPerPage: p.Count,
	}

	for _, user := range users {
		result.Resources = append(result.Resources, *user.ToAPI())
	}

	return result, nil
}
