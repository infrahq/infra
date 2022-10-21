package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/scim2/filter-parser/v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

var getProviderUsersRoute = route[api.Resource, *api.SCIMUser]{
	handler: GetProviderUser,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

var listProviderUsersRoute = route[api.SCIMParametersRequest, *api.ListProviderUsersResponse]{
	handler: ListProviderUsers,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

var createProviderUserRoute = route[api.SCIMUserCreateRequest, *api.SCIMUser]{
	handler: CreateProviderUser,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

var updateProviderUserRoute = route[api.SCIMUserUpdateRequest, *api.SCIMUser]{
	handler: UpdateProviderUser,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

var patchProviderUserRoute = route[api.SCIMUserPatchRequest, *api.SCIMUser]{
	handler: PatchProviderUser,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

var deleteProviderUserRoute = route[api.Resource, *api.EmptyResponse]{
	handler: DeleteProviderUser,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
	},
}

func GetProviderUser(c *gin.Context, r *api.Resource) (*api.SCIMUser, error) {
	user, err := access.GetProviderUser(c, r.ID)
	if err != nil {
		return nil, err
	}
	return user.ToAPI(), nil
}

func ListProviderUsers(c *gin.Context, r *api.SCIMParametersRequest) (*api.ListProviderUsersResponse, error) {
	p := data.SCIMParameters{
		StartIndex: r.StartIndex,
		Count:      r.Count,
	}
	if r.Filter != "" {
		exp, err := filter.ParseFilter([]byte(r.Filter))
		if err != nil {
			return nil, fmt.Errorf("parse SCIM filter expression: %w", err)
		}
		p.Filter = exp
	}
	users, err := access.ListProviderUsers(c, &p)
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

func CreateProviderUser(c *gin.Context, r *api.SCIMUserCreateRequest) (*api.SCIMUser, error) {
	user := &models.ProviderUser{
		GivenName:  r.Name.GivenName,
		FamilyName: r.Name.FamilyName,
		Active:     r.Active,
	}
	for _, email := range r.Emails {
		if email.Primary {
			user.Email = email.Value
		}
	}
	if user.Email == "" {
		return nil, fmt.Errorf("%w: primary email is required", internal.ErrBadRequest)
	}
	err := access.CreateProviderUser(c, user)
	if err != nil {
		return nil, err
	}
	return user.ToAPI(), nil
}

func UpdateProviderUser(c *gin.Context, r *api.SCIMUserUpdateRequest) (*api.SCIMUser, error) {
	user := &models.ProviderUser{
		IdentityID: r.ID,
		GivenName:  r.Name.GivenName,
		FamilyName: r.Name.FamilyName,
		Active:     r.Active,
	}
	for _, email := range r.Emails {
		if email.Primary {
			user.Email = email.Value
		}
	}
	if user.Email == "" {
		return nil, fmt.Errorf("%w: primary email is required", internal.ErrBadRequest)
	}
	err := access.UpdateProviderUser(c, user)
	if err != nil {
		return nil, err
	}
	return user.ToAPI(), nil
}

func PatchProviderUser(c *gin.Context, r *api.SCIMUserPatchRequest) (*api.SCIMUser, error) {
	// we only support active status patching, so there can only be one operation
	if len(r.Operations) != 1 || r.Operations[0].Op != "replace" {
		return nil, internal.ErrBadRequest
	}

	user := &models.ProviderUser{
		IdentityID: r.ID,
		Active:     r.Operations[0].Value.Active,
	}
	result, err := access.PatchProviderUser(c, user)
	if err != nil {
		return nil, err
	}
	return result.ToAPI(), nil
}

func DeleteProviderUser(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteProviderUser(c, r.ID)
}
