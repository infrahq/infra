package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
)

func (a *API) RequestForgotDomains(c *gin.Context, r *api.ForgotDomainRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)

	if err := redis.NewLimiter(a.server.redis).RateOK(r.Email, 10); err != nil {
		return nil, err
	}

	domains, err := data.GetForgottenDomainsForEmail(rCtx.DBTxn, r.Email)
	switch {
	case err != nil:
		return nil, err
	case len(domains) == 0:
		return nil, nil // This is okay. we don't notify the user if we failed to find the email.
	}

	err = email.SendForgotDomainsEmail("", r.Email, email.ForgottenDomainData{Domains: domains})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
