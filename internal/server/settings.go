package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
)

func (a *API) GetSettings(c *gin.Context, r *api.EmptyRequest) (*api.Settings, error) {
	settings, err := access.GetSettings(c)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	return settings.ToAPI(), nil
}
