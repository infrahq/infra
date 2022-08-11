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

func initializeSettings(db *gorm.DB) (*models.Settings, error) {
	org := OrgFromContext(db.Statement.Context)

	settings, err := GetSettings(db)
	if settings != nil {
		return settings, err
	}

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

	settings = &models.Settings{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		PrivateJWK:         secs,
		PublicJWK:          pubs,
	}

	// Attrs() assigns the field iff the record is not found
	if err := db.Where("organization_id = ?", org.ID).FirstOrCreate(&settings).Error; err != nil {
		return nil, err
	}

	return settings, nil
}

func GetSettings(db *gorm.DB) (*models.Settings, error) {
	org := OrgFromContext(db.Statement.Context)

	var settings models.Settings
	if err := db.Where("organization_id = ?", org.ID).First(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func SaveSettings(db *gorm.DB, settings *models.Settings) error {
	// TODO: clean this up by having the query use the organization_id instead of the
	// primary key in the WHERE.
	existing, err := GetSettings(db)
	if err != nil {
		return err
	}
	settings.ID = existing.ID

	return save(db, settings)
}
