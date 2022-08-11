package models

import (
	"github.com/infrahq/infra/api"
)

type Settings struct {
	Model
	OrganizationMember

	PrivateJWK EncryptedAtRestBytes
	PublicJWK  []byte

	LowercaseMin int `gorm:"default:0"`
	UppercaseMin int `gorm:"default:0"`
	NumberMin    int `gorm:"default:0"`
	SymbolMin    int `gorm:"default:0"`
	LengthMin    int `gorm:"default:8"`
}

func (s *Settings) ToAPI() *api.Settings {
	return &api.Settings{
		PasswordRequirements: api.PasswordRequirements{
			LowercaseMin: s.LowercaseMin,
			UppercaseMin: s.UppercaseMin,
			NumberMin:    s.NumberMin,
			SymbolMin:    s.SymbolMin,
			LengthMin:    s.LengthMin,
		},
	}
}

func (s *Settings) SetFromAPI(a *api.Settings) {
	s.LengthMin = a.PasswordRequirements.LowercaseMin
	s.UppercaseMin = a.PasswordRequirements.UppercaseMin
	s.LowercaseMin = a.PasswordRequirements.LowercaseMin
	s.SymbolMin = a.PasswordRequirements.SymbolMin
	s.NumberMin = a.PasswordRequirements.NumberMin
}
