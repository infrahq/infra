package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

// IdentityKind determines the use case of an identitiy, either a user or machine
type IdentityKind string

const (
	UserKind    IdentityKind = "user"
	MachineKind IdentityKind = "machine"
)

func (ik IdentityKind) String() string {
	return string(ik)
}

func ParseIdentityKind(s string) (IdentityKind, error) {
	kinds := map[IdentityKind]bool{
		UserKind:    true,
		MachineKind: true,
	}

	s = strings.ToLower(s)

	kind := IdentityKind(s)

	_, ok := kinds[kind]
	if !ok {
		return kind, fmt.Errorf(`invalid identity kind %q`, s)
	}

	return kind, nil
}

type Identity struct {
	Model

	Kind       IdentityKind
	Name       string    `gorm:"uniqueIndex:idx_identities_name_provider_id,where:deleted_at is NULL"`
	LastSeenAt time.Time // updated on when an identity uses a session token

	ProviderID uid.ID `gorm:"uniqueIndex:idx_identities_name_provider_id,where:deleted_at is NULL"`

	Groups []Group `gorm:"many2many:identities_groups"`
}

func (i *Identity) ToAPI() *api.Identity {
	return &api.Identity{
		ID:         i.ID,
		Created:    api.Time(i.CreatedAt),
		Updated:    api.Time(i.UpdatedAt),
		LastSeenAt: api.Time(i.LastSeenAt),
		Name:       i.Name,
		Kind:       i.Kind.String(),
		ProviderID: i.ProviderID,
	}
}

// PolyID is a polymorphic name that points to both a model type and an ID
func (i *Identity) PolyID() uid.PolymorphicID {
	return uid.NewIdentityPolymorphicID(i.ID)
}
