package server

import (
	"fmt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/email"
)

func (a *API) GetServerConfiguration(rCtx access.RequestContext, _ *api.EmptyRequest) (*api.ServerConfiguration, error) {
	config := &api.ServerConfiguration{
		IsEmailConfigured: email.IsConfigured(),
		BaseDomain:        a.server.options.BaseDomain,
		LoginDomain:       a.server.options.BaseDomain, // default to the standard base domain
	}
	if a.server.options.LoginDomainPrefix != "" {
		config.LoginDomain = fmt.Sprintf("%s.%s", a.server.options.LoginDomainPrefix, a.server.options.BaseDomain)
	}
	if a.server.Google != nil {
		config.Google = a.server.Google.ToAPI()
	}

	return config, nil
}
