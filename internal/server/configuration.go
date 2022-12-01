package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/email"
)

func (a *API) GetServerConfiguration(c *gin.Context, _ *api.EmptyRequest) (*api.ServerConfiguration, error) {
	conf := &api.ServerConfiguration{
		IsEmailConfigured: email.IsConfigured(),
		BaseDomain:        a.server.options.BaseDomain,
		LoginDomain:       a.server.options.BaseDomain, // default to the standard base domain
	}
	if a.server.options.LoginDomainPrefix != "" {
		conf.LoginDomain = fmt.Sprintf("%s.%s", a.server.options.LoginDomainPrefix, a.server.options.BaseDomain)
	}
	return conf, nil
}
