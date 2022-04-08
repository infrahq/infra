package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

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
