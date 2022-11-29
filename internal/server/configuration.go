package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/email"
)

func (a *API) GetServerConfiguration(c *gin.Context, _ *api.EmptyRequest) (*api.ServerConfiguration, error) {
	return &api.ServerConfiguration{
		IsEmailConfigured: email.IsConfigured(),
		BaseDomain:        a.server.options.BaseDomain,
		LoginRedirect:     a.server.options.LoginRedirect,
	}, nil
}
