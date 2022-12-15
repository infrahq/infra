package api

import "github.com/infrahq/infra/internal/validate"

type Settings struct {
	PasswordRequirements PasswordRequirements `json:"passwordRequirements"`
}

type PasswordRequirements struct {
	LengthMin    int `json:"lengthMin" note:"Minimum password length. Must be at least 8 characters."`
	LowercaseMin int `json:"lowercaseMin" note:"Minimum number of lowercase ASCII letters."`
	UppercaseMin int `json:"uppercaseMin" note:"Minimum number of uppercase ASCII letters."`
	NumberMin    int `json:"numberMin" note:"Minimum number of numbers."`
	SymbolMin    int `json:"symbolMin" note:"Minimum number of symbols."`
}

func (s Settings) ValidationRules() []validate.ValidationRule {
	return nil
}

func (r PasswordRequirements) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.IntRule{Name: "lengthMin", Value: r.LengthMin, Min: validate.Int(8)},
	}
}
