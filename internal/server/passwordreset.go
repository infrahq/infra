package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
)

func (a *API) RequestPasswordReset(rCtx access.RequestContext, r *api.PasswordResetRequest) (*api.EmptyResponse, error) {
	// no authorization required
	if err := redis.NewLimiter(a.server.redis).RateOK(r.Email, 10); err != nil {
		return nil, err
	}

	user, err := data.GetIdentity(rCtx.DBTxn, data.GetIdentityOptions{ByName: r.Email})
	switch {
	case errors.Is(err, internal.ErrNotFound):
		return nil, nil // This is okay. we don't notify the user if we failed to find the email.
	case err != nil:
		return nil, err
	}

	_, err = data.GetCredentialByUserID(rCtx.DBTxn, user.ID)
	if err != nil {
		// if credential is not found, the user cannot reset their password.
		return nil, err
	}

	token, err := data.CreatePasswordResetToken(rCtx.DBTxn, user.ID, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	org := rCtx.Authenticated.Organization
	err = email.SendPasswordResetEmail("", r.Email, email.PasswordResetData{
		Link: wrapLinkWithVerification(fmt.Sprintf("https://%s/password-reset?token=%s", org.Domain, token), org.Domain, user.VerificationToken),
	})

	return nil, err
}

func (a *API) VerifiedPasswordReset(rCtx access.RequestContext, r *api.VerifiedResetPasswordRequest) (*api.LoginResponse, error) {
	user, err := access.VerifiedPasswordReset(rCtx, r.Token, r.Password)
	if err != nil {
		return nil, err
	}

	return a.Login(rCtx, &api.LoginRequest{
		PasswordCredentials: &api.LoginRequestPasswordCredentials{
			Name:     user.Name,
			Password: r.Password,
		},
	})
}
