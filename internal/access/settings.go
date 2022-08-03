package access

import (
	"fmt"

	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func GetPublicJWK(c RequestContext) ([]jose.JSONWebKey, error) {
	settings, err := data.GetSettings(c.DBTxn)
	if err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	var pubKey jose.JSONWebKey
	if err := pubKey.UnmarshalJSON(settings.PublicJWK); err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	return []jose.JSONWebKey{pubKey}, nil
}

func GetSettings(c *gin.Context) (*models.Settings, error) {
	db := getDB(c)
	return data.GetSettings(db)
}

func SaveSettings(c *gin.Context, settings *models.Settings) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "settings", "update", models.InfraAdminRole)
	}

	if err = data.SaveSettings(db, settings); err != nil {
		return err
	}
	return nil
}
