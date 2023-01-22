package server

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
)

func (a *API) RequestForgotDomains(rCtx access.RequestContext, r *api.ForgotDomainRequest) (*api.EmptyResponse, error) {
	

	if err := redis.NewLimiter(a.server.redis).RateOK(r.Email, 10); err != nil {
		return nil, err
	}

	orgs, err := data.GetForgottenDomainsForEmail(rCtx.DBTxn, r.Email)
	switch {
	case err != nil:
		return nil, err
	case len(orgs) == 0:
		return nil, nil // This is okay. we don't notify the user if we failed to find the email.
	}

	err = email.SendForgotDomainsEmail("", r.Email, email.ForgottenDomainData{Organizations: orgs})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
