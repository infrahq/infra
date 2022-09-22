package server

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
)

func (a *API) RequestForgotDomains(c *gin.Context, r *api.ForgotDomainRequest) (*api.EmptyResponse, error) {
	if err := redis.RateOK(a.server.redis, r.Email, 10); err != nil {
		return nil, err
	}

	domains, err := access.ForgotDomainRequest(c, r.Email)

	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return nil, nil // This is okay. we don't notify the user if we failed to find the email.
		}
		return nil, err
	}

	err = email.SendForgotDomainsEmail("", r.Email, email.ForgottenDomainData{Domains: domains})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
