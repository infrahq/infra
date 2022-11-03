package data

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"

	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type settingsTable models.Settings

func (settingsTable) Table() string {
	return "settings"
}

func (s settingsTable) Columns() []string {
	return []string{"created_at", "deleted_at", "id", "length_min", "lowercase_min", "number_min", "organization_id", "private_jwk", "public_jwk", "symbol_min", "updated_at", "uppercase_min"}
}

func (s settingsTable) Values() []any {
	return []any{s.CreatedAt, s.DeletedAt, s.ID, s.LengthMin, s.LowercaseMin, s.NumberMin, s.OrganizationID, s.PrivateJWK, s.PublicJWK, s.SymbolMin, s.UpdatedAt, s.UppercaseMin}
}

func (s *settingsTable) ScanFields() []any {
	return []any{&s.CreatedAt, &s.DeletedAt, &s.ID, &s.LengthMin, &s.LowercaseMin, &s.NumberMin, &s.OrganizationID, &s.PrivateJWK, &s.PublicJWK, &s.SymbolMin, &s.UpdatedAt, &s.UppercaseMin}
}

func createSettings(tx WriteTxn, orgID uid.ID) error {
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

	settings := &models.Settings{
		OrganizationMember: models.OrganizationMember{OrganizationID: orgID},
		PrivateJWK:         models.EncryptedAtRest(secs),
		PublicJWK:          pubs,
		LengthMin:          8,
	}
	return insert(tx, (*settingsTable)(settings))
}

func GetSettings(db ReadTxn) (*models.Settings, error) {
	return getSettingsForOrg(db, db.OrganizationID())
}

func getSettingsForOrg(tx ReadTxn, orgID uid.ID) (*models.Settings, error) {
	settings := settingsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(settings))
	query.B("FROM settings")
	query.B("WHERE deleted_at is null AND organization_id = ?", orgID)

	err := tx.QueryRow(query.String(), query.Args...).Scan(settings.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.Settings)(&settings), nil
}

func UpdateSettings(tx WriteTxn, settings *models.Settings) error {
	if settings.ID == 0 {
		existing, err := GetSettings(tx)
		if err != nil {
			return err
		}
		settings.ID = existing.ID
	}
	return update(tx, (*settingsTable)(settings))
}
