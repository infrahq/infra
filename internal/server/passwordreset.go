package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
)

func (a *API) RequestPasswordReset(c *gin.Context, r *api.PasswordResetRequest) (*api.EmptyResponse, error) {
	if err := redis.NewLimiter(a.server.redis).RateOK(r.Email, 10); err != nil {
		return nil, err
	}

	token, user, err := access.PasswordResetRequest(c, r.Email, 15*time.Minute)
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return nil, nil // This is okay. we don't notify the user if we failed to find the email.
		}
		return nil, err
	}

	org := access.GetRequestContext(c).Authenticated.Organization

	// send email
	err = email.SendPasswordResetEmail("", r.Email, email.PasswordResetData{
		Link: wrapLinkWithVerification(fmt.Sprintf("https://%s/password-reset?token=%s", org.Domain, token), org.Domain, user.VerificationToken),
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
