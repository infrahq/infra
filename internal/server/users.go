package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListUsers(c *gin.Context, r *api.ListUsersRequest) (*api.ListResponse[api.User], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	users, err := access.ListIdentities(c, r.Name, r.Group, r.IDs, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(users, models.PaginationToResponse(p), func(identity models.Identity) api.User {
		return *identity.ToAPI()
	})

	return result, nil
}

func (a *API) GetUser(c *gin.Context, r *api.GetUserRequest) (*api.User, error) {
	if r.ID.IsSelf {
		iden := access.AuthenticatedIdentity(c)
		if iden == nil {
			return nil, fmt.Errorf("%w: no user is logged in", internal.ErrUnauthorized)
		}
		r.ID.ID = iden.ID
	}
	identity, err := access.GetIdentity(c, r.ID.ID)
	if err != nil {
		return nil, err
	}

	return identity.ToAPI(), nil
}

// CreateUser creates a user with the Infra provider
func (a *API) CreateUser(c *gin.Context, r *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	user := &models.Identity{Name: r.Name}
	infraProvider := access.InfraProvider(c)

	// infra identity creation should be attempted even if an identity is already known
	identities, err := access.ListIdentities(c, user.Name, 0, nil, &models.Pagination{Limit: 2})
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}

	switch len(identities) {
	case 0:
		if err := access.CreateIdentity(c, user); err != nil {
			return nil, fmt.Errorf("create identity: %w", err)
		}
	case 1:
		user.ID = identities[0].ID
	default:
		logging.Errorf("Multiple identites match name %q. DB is missing unique index on user names", r.Name)
		return nil, fmt.Errorf("multiple identities match specified name") // should not happen
	}

	resp := &api.CreateUserResponse{
		ID:   user.ID,
		Name: user.Name,
	}

	_, err = access.CreateProviderUser(c, infraProvider, user)
	if err != nil {
		return nil, fmt.Errorf("creating provider user: %w", err)
	}

	// Always create a temporary password for infra users.
	tmpPassword, err := access.CreateCredential(c, *user)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	resp.OneTimePassword = tmpPassword

	return resp, nil
}

func (a *API) UpdateUser(c *gin.Context, r *api.UpdateUserRequest) (*api.User, error) {
	// right now this endpoint can only update a user's credentials, so get the user identity
	identity, err := access.GetIdentity(c, r.ID)
	if err != nil {
		return nil, err
	}

	err = access.UpdateCredential(c, identity, r.Password)
	if err != nil {
		return nil, err
	}

	// if the user is an admin, we could be required to create the infra user, so create the provider_user if it's missing.
	_, _ = access.CreateProviderUser(c, access.InfraProvider(c), identity)

	return identity.ToAPI(), nil
}

func (a *API) DeleteUser(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteIdentity(c, r.ID)
}
