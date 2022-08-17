package server

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
)

func (a *API) RequestPasswordReset(c *gin.Context, r *api.PasswordResetRequest) (*api.EmptyResponse, error) {
	// TODO: rate-limit

	token, err := access.PasswordResetRequest(c, r.Email)
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return nil, nil // This is okay. we don't notify the user if we failed to find the email.
		}
		return nil, err
	}

	org := data.MustGetOrgFromContext(c)

	// send email
	err = email.SendPasswordReset("", r.Email, email.PasswordResetData{
		Link: fmt.Sprintf("https://%s/password-reset?token=%s", org.Domain, token),
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *API) VerifiedPasswordReset(c *gin.Context, r *api.VerifiedResetPasswordRequest) (*api.LoginResponse, error) {
	user, err := access.VerifiedPasswordReset(c, r.Token, r.Password)
	if err != nil {
		return nil, err
	}

	return a.Login(c, &api.LoginRequest{
		PasswordCredentials: &api.LoginRequestPasswordCredentials{
			Name:     user.Name,
			Password: r.Password,
		},
	})
}
