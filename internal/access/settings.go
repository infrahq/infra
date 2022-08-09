package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func GetPublicJWK(c RequestContext) ([]jose.JSONWebKey, error) {
	org, ok := c.DBTxn.Statement.Context.Value(data.OrgCtxKey{}).(*models.Organization)
	if !ok {
		return nil, errors.New("unknown organization")
	}

	settings, err := data.GetSettings(c.DBTxn, org)
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
	org, ok := db.Statement.Context.Value(data.OrgCtxKey{}).(*models.Organization)
	if !ok {
		return nil, errors.New("unknown organization")
	}

	return data.GetSettings(db, org)
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
