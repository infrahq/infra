package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
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

func GetPasswordSettings(c *gin.Context) (api.PasswordResponse, error) {
	db := getDB(c)
	settings, err := data.GetSettings(db)
	if err != nil {
		return api.PasswordResponse{}, fmt.Errorf("could not get password settings: %w", err)
	}

	var pwSettings api.PasswordResponse
	pwSettings.LowercaseMin = settings.LowercaseMin
	pwSettings.UppercaseMin = settings.UppercaseMin
	pwSettings.NumberMin = settings.NumberMin
	pwSettings.SymbolMin = settings.SymbolMin
	pwSettings.LengthMin = settings.LengthMin
	return pwSettings, nil
}
