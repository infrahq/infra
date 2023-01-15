package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/server/data"

	"github.com/infrahq/infra/api"
)

func (a *API) GetSettings(c *gin.Context, r *api.EmptyRequest) (*api.Settings, error) {
	// No authorization required
	rCtx := getRequestContext(c)
	settings, err := data.GetSettings(rCtx.DBTxn)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	return settings.ToAPI(), nil
}
