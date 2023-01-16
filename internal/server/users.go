package server

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
)

func (a *API) ListUsers(c *gin.Context, r *api.ListUsersRequest) (*api.ListResponse[api.User], error) {
	p := PaginationFromRequest(r.PaginationRequest)

	opts := data.ListIdentityOptions{
		Pagination:             &p,
		ByName:                 r.Name,
		ByIDs:                  r.IDs,
		ByGroupID:              r.Group,
		ByPublicKeyFingerprint: r.PublicKeyFingerprint,
		LoadProviders:          true,
		LoadPublicKeys:         r.PublicKeyFingerprint != "",
	}
	if !r.ShowSystem {
		opts.ByNotName = models.InternalInfraConnectorIdentityName
	}

	users, err := access.ListIdentities(c, opts)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(users, PaginationToResponse(p), func(identity models.Identity) api.User {
		return *identity.ToAPI()
	})

	return result, nil
}

var getUserRoute = route[api.GetUserRequest, *api.User]{
	routeSettings: routeSettings{
		omitFromTelemetry: true,
		txnOptions:        &sql.TxOptions{ReadOnly: true},
		// the UI calls this endpoint to check session status
		idpSync: true,
	},
	handler: GetUser,
}

func GetUser(c *gin.Context, r *api.GetUserRequest) (*api.User, error) {
	if r.ID.IsSelf {
		iden := access.GetRequestContext(c).Authenticated.User
		if iden == nil {
			return nil, fmt.Errorf("no authenticated user")
		}
		r.ID.ID = iden.ID
	}
	identity, err := access.GetIdentity(c, data.GetIdentityOptions{
		ByID:           r.ID.ID,
		LoadProviders:  true,
		LoadPublicKeys: true,
	})
	if err != nil {
		return nil, err
	}

	return identity.ToAPI(), nil
}

// CreateUser creates a user with the Infra provider
func (a *API) CreateUser(c *gin.Context, r *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	user := &models.Identity{Name: r.Name}

	// infra identity creation should be attempted even if an identity is already known
	identities, err := access.ListIdentities(c, data.ListIdentityOptions{ByName: r.Name})
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
		logging.Errorf("Multiple identities match name %q. DB is missing unique index on user names", r.Name)
		return nil, fmt.Errorf("multiple identities match specified name") // should not happen
	}

	resp := &api.CreateUserResponse{
		ID:   user.ID,
		Name: user.Name,
	}

	// Always create a temporary password for infra users.
	tmpPassword, err := access.CreateCredential(c, user)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	if email.IsConfigured() {
		rCtx := access.GetRequestContext(c)
		org := rCtx.Authenticated.Organization
		currentUser := rCtx.Authenticated.User

		// hack because we don't have names.
		fromName := email.BuildNameFromEmail(currentUser.Name)

		token, err := data.CreatePasswordResetToken(rCtx.DBTxn, user.ID, 72*time.Hour)
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

func (a *API) UpdateUser(c *gin.Context, r *api.UpdateUserRequest) (*api.UpdateUserResponse, error) {
	rCtx := access.GetRequestContext(c)

	if rCtx.Authenticated.User.ID == r.ID {
		if err := access.UpdateCredential(c, rCtx.Authenticated.User, r.OldPassword, r.Password); err != nil {
			return nil, err
		}

		return &api.UpdateUserResponse{
			User: *rCtx.Authenticated.User.ToAPI(),
		}, nil
	}

	user, err := access.GetIdentity(c, data.GetIdentityOptions{ByID: r.ID, LoadProviders: true})
	if err != nil {
		return nil, err
	}

	password, err := access.ResetCredential(c, user, r.Password)
	if err != nil {
		return nil, err
	}

	return &api.UpdateUserResponse{
		User:            *user.ToAPI(),
		OneTimePassword: password,
	}, nil
}

func (a *API) DeleteUser(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	rCtx := access.GetRequestContext(c)
	if rCtx.Authenticated.User.ID == r.ID {
		return nil, fmt.Errorf("%w: cannot delete own user", internal.ErrBadRequest)
	}

	if data.InfraConnectorIdentity(rCtx.DBTxn).ID == r.ID {
		return nil, fmt.Errorf("%w: cannot delete connector user", internal.ErrBadRequest)
	}

	return nil, access.DeleteIdentity(c, r.ID)
}

func AddUserPublicKey(c *gin.Context, r *api.AddUserPublicKeyRequest) (*api.UserPublicKey, error) {
	rCtx := getRequestContext(c)

	// no authz required, because the userID comes from authenticated User.ID
	if rCtx.Authenticated.User == nil {
		return nil, fmt.Errorf("missing authentication")
	}

	key, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(r.PublicKey))
	switch {
	case err != nil:
		// the error text is always the same "ssh: no key found", so we return
		// a better error message.
		return nil, validate.Error{"publicKey": {"must be in authorized_keys format"}}
	case len(bytes.TrimSpace(rest)) > 0:
		return nil, validate.Error{"publicKey": {"must be only a single key"}}
	}

	userPublicKey := &models.UserPublicKey{
		Name:        r.Name,
		UserID:      rCtx.Authenticated.User.ID,
		PublicKey:   base64.StdEncoding.EncodeToString(key.Marshal()),
		KeyType:     key.Type(),
		Fingerprint: ssh.FingerprintSHA256(key),
		ExpiresAt:   time.Now().Add(12 * time.Hour),
	}

	if err := data.AddUserPublicKey(rCtx.DBTxn, userPublicKey); err != nil {
		return nil, err
	}
	resp := userPublicKey.ToAPI()
	return &resp, nil
}
