package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListUsers(c *gin.Context, r *api.ListUsersRequest) (*api.ListResponse[api.User], error) {
	p := PaginationFromRequest(r.PaginationRequest)
	users, err := access.ListIdentities(c, r.Name, r.Group, r.IDs, r.ShowSystem, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(users, PaginationToResponse(p), func(identity models.Identity) api.User {
		return *identity.ToAPI()
	})

	return result, nil
}

func (a *API) GetUser(c *gin.Context, r *api.GetUserRequest) (*api.User, error) {
	if r.ID.IsSelf {
		iden := access.GetRequestContext(c).Authenticated.User
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

	// infra identity creation should be attempted even if an identity is already known
	identities, err := access.ListIdentities(c, user.Name, 0, nil, false, &data.Pagination{Limit: 2})
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

	// Always create a temporary password for infra users.
	tmpPassword, err := access.CreateCredential(c, *user)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	if email.IsConfigured() {
		rCtx := access.GetRequestContext(c)
		org := rCtx.Authenticated.Organization
		currentUser := rCtx.Authenticated.User

		// hack because we don't have names.
		fromName := email.BuildNameFromEmail(currentUser.Name)

		token, user, err := access.PasswordResetRequest(c, user.Name, 72*time.Hour)
		if err != nil {
			return nil, err
		}

		err = email.SendUserInviteEmail("", user.Name, email.UserInviteData{
			FromUserName: fromName,
			Link:         fmt.Sprintf("https://%s/accept-invite?token=%s", org.Domain, token),
		})
		if err != nil {
			return nil, fmt.Errorf("sending invite email: %w", err)
		}
	} else {
		resp.OneTimePassword = tmpPassword
	}

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
	return identity.ToAPI(), nil
}

func (a *API) DeleteUser(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteIdentity(c, r.ID)
}
