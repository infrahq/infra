package server

import (
	"errors"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
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

	// TODO: adjust domain for orgs when here
	link, _ := url.Parse(email.AppDomain)
	link.Path = "/password-reset"
	query := link.Query()
	query.Add("token", token)
	link.RawQuery = query.Encode()

	// send email
	err = email.SendPasswordReset("", r.Email, email.PasswordResetData{
		Link: link.String(),
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
