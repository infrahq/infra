package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
)

func (a *API) GetSettings(c *gin.Context, r *api.EmptyRequest) (*api.Settings, error) {
	return access.GetSettings(c)
}

func (a *API) UpdateSettings(c *gin.Context, s *api.Settings) (*api.Settings, error) {
	return access.SaveSettings(c, s)
}
