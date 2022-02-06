package data

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"

	"gopkg.in/square/go-jose.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

func InitializeSettings(db *gorm.DB) (*models.Settings, error) {
	pubkey, seckey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sec := jose.JSONWebKey{Key: seckey, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}

	thumb, err := sec.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, err
	}

	sec.KeyID = base64.URLEncoding.EncodeToString(thumb)

	pub := jose.JSONWebKey{Key: pubkey, KeyID: sec.KeyID, Algorithm: string(jose.ED25519), Use: "sig"}

	secs, err := sec.MarshalJSON()
	if err != nil {
		return nil, err
	}

	pubs, err := pub.MarshalJSON()
	if err != nil {
		return nil, err
	}

	settings := models.Settings{
		PrivateJWK: secs,
		PublicJWK:  pubs,
	}

	if err := db.FirstOrCreate(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func GetSettings(db *gorm.DB) (*models.Settings, error) {
	var settings models.Settings
	if err := db.First(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}
