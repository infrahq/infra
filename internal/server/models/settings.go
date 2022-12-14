package models

import (
	"github.com/infrahq/infra/api"
)

type Settings struct {
	Model
	OrganizationMember

	PrivateJWK EncryptedAtRest
	PublicJWK  []byte

	LowercaseMin int
	UppercaseMin int
	NumberMin    int
	SymbolMin    int
	LengthMin    int
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
	s.LengthMin = a.PasswordRequirements.LengthMin
	s.UppercaseMin = a.PasswordRequirements.UppercaseMin
	s.LowercaseMin = a.PasswordRequirements.LowercaseMin
	s.SymbolMin = a.PasswordRequirements.SymbolMin
	s.NumberMin = a.PasswordRequirements.NumberMin
}
