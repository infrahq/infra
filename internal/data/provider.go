package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
)

type ProviderKind string

var ProviderKindOkta ProviderKind = "okta"

type Provider struct {
	Model

	Kind ProviderKind

	Domain       string
	ClientID     string
	ClientSecret string

	Okta ProviderOkta

	Users  []User  `gorm:"many2many:users_providers"`
	Groups []Group `gorm:"many2many:groups_providers"`
}

type ProviderOkta struct {
	Model

	APIToken string

	ProviderID uuid.UUID
}

func (p *Provider) ToAPI() api.Provider {
	result := api.Provider{
		Id:      p.ID.String(),
		Created: p.CreatedAt.Unix(),
		Updated: p.UpdatedAt.Unix(),

		Kind:     api.ProviderKind(p.Kind),
		Domain:   p.Domain,
		ClientID: p.ClientID,
	}

	// switch p.Kind {
	// case ProviderKindOkta:
	// }

	// 	for _, u := range p.Users {
	// 		result.Users = append(result.Users, u.ToAPI())
	// 	}

	// 	for _, g := range p.Groups {
	// 		result.Groups = append(result.Groups, g.ToAPI())
	// 	}

	return result
}

func NewProvider(id string) (*Provider, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Provider{
		Model: Model{
			ID: uid,
		},
	}, nil
}

func CreateProvider(db *gorm.DB, provider *Provider) (*Provider, error) {
	if err := add(db, &Provider{}, provider, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func CreateOrUpdateProvider(db *gorm.DB, provider *Provider, condition interface{}) (*Provider, error) {
	existing, err := GetProvider(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateProvider(db, provider); err != nil {
			return nil, err
		}

		return provider, nil
	}

	if err := update(db, &Provider{}, provider, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch provider.Kind {
	case ProviderKindOkta:
		if err := db.Model(existing).Association("Okta").Replace(&provider.Okta); err != nil {
			return nil, err
		}
	}

	return GetProvider(db, db.Where(existing, "id"))
}

func GetProvider(db *gorm.DB, condition interface{}) (*Provider, error) {
	var provider Provider
	if err := get(db, &Provider{}, &provider, condition); err != nil {
		return nil, err
	}

	return &provider, nil
}

func ListProviders(db *gorm.DB, condition interface{}) ([]Provider, error) {
	providers := make([]Provider, 0)
	if err := list(db, &Provider{}, &providers, condition); err != nil {
		return nil, err
	}

	return providers, nil
}

func DeleteProviders(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListProviders(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &Provider{}, ids)
	}

	return nil
}
