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

func GetSettings(c *gin.Context) (*api.Settings, error) {
	db := getDB(c)
	settings, err := data.GetSettings(db)
	if err != nil {
		return &api.Settings{}, fmt.Errorf("could not get settings: %w", err)
	}

	return settings.ToAPI(), nil
}

func SaveSettings(c *gin.Context, updatedSettings *api.Settings) (*api.Settings, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "settings", "update", models.InfraAdminRole)
	}

	settings, err := data.GetSettings(db)
	if err != nil {
		return nil, err
	}

	settings.SetFromAPI(updatedSettings)
	if err = data.SaveSettings(db, settings); err != nil {
		return nil, err
	}

	settings, err = data.GetSettings(db)
	if err != nil {
		return nil, fmt.Errorf("could not get settings after update: %w", err)
	}
	return settings.ToAPI(), nil
}
