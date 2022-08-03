package server

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) GetSettings(c *gin.Context, r *api.GetSettingsRequest) (*api.Settings, error) {
	res := api.Settings{}

	if r.PasswordRequirements {
		requirements, err := access.GetPasswordRequirements(c)
		if err != nil {
			return nil, err
		}
		res.PasswordRequirements = requirements
	}

	return &res, nil
}

func (a *API) UpdateSettings(c *gin.Context, r *api.Settings) (*api.EmptyResponse, error) {
	settings := &models.Settings{
		LowercaseMin: r.PasswordRequirements.LowercaseMin,
		UppercaseMin: r.PasswordRequirements.UppercaseMin,
		SymbolMin:    r.PasswordRequirements.SymbolMin,
		NumberMin:    r.PasswordRequirements.NumberMin,
		LengthMin:    r.PasswordRequirements.LengthMin,
	}

	return nil, access.SaveSettings(c, settings)
}
