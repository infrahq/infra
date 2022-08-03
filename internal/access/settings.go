package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func GetPublicJWK(c *gin.Context) ([]jose.JSONWebKey, error) {
	db := getDB(c)
	settings, err := data.GetSettings(db)
	if err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	var pubKey jose.JSONWebKey
	if err := pubKey.UnmarshalJSON(settings.PublicJWK); err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	return []jose.JSONWebKey{pubKey}, nil
}

func GetPasswordRequirements(c *gin.Context) (api.PasswordRequirements, error) {
	db := getDB(c)
	settings, err := data.GetSettings(db)

	if err != nil {
		return api.PasswordRequirements{}, fmt.Errorf("could not get password settings: %w", err)
	}

	var requirements api.PasswordRequirements
	requirements.LowercaseMin = settings.LowercaseMin
	requirements.UppercaseMin = settings.UppercaseMin
	requirements.NumberMin = settings.NumberMin
	requirements.SymbolMin = settings.SymbolMin
	requirements.LengthMin = settings.LengthMin
	return requirements, nil
}

func SaveSettings(c *gin.Context, updatedSettings *models.Settings) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "settings", "update", models.InfraAdminRole)
	}

	settings, err := data.GetSettings(db)
	if err != nil {
		return err
	}
	settings.LengthMin = updatedSettings.LengthMin
	settings.UppercaseMin = updatedSettings.UppercaseMin
	settings.LowercaseMin = updatedSettings.LowercaseMin
	settings.SymbolMin = updatedSettings.SymbolMin
	settings.NumberMin = updatedSettings.NumberMin

	return data.SaveSettings(db, settings)
}
