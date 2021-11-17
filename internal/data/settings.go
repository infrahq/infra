package data

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"

	"gopkg.in/square/go-jose.v2"
	"gorm.io/gorm"
)

type Settings struct {
	Model

	PrivateJWK []byte
	PublicJWK  []byte
}

func (s *Settings) BeforeSave(tx *gorm.DB) error {
	if len(s.PrivateJWK) != 0 && len(s.PublicJWK) != 0 {
		return nil
	}

	pubkey, seckey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	sec := jose.JSONWebKey{Key: seckey, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}

	thumb, err := sec.Thumbprint(crypto.SHA256)
	if err != nil {
		return err
	}

	sec.KeyID = base64.URLEncoding.EncodeToString(thumb)

	pub := jose.JSONWebKey{Key: pubkey, KeyID: sec.KeyID, Algorithm: string(jose.ED25519), Use: "sig"}

	secs, err := sec.MarshalJSON()
	if err != nil {
		return err
	}

	pubs, err := pub.MarshalJSON()
	if err != nil {
		return err
	}

	s.PrivateJWK = secs
	s.PublicJWK = pubs

	return nil
}

func InitializeSettings(db *gorm.DB) (*Settings, error) {
	var settings Settings
	if err := db.FirstOrCreate(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func GetSettings(db *gorm.DB) (*Settings, error) {
	var settings Settings
	if err := db.First(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}
