package models

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
)

func TestSettingsSetFromAPI(t *testing.T) {
	type testCase struct {
		api.PasswordRequirements
		Settings
	}

	cases := map[string]testCase{
		"length": {
			PasswordRequirements: api.PasswordRequirements{
				LengthMin: 16,
			},
			Settings: Settings{
				LengthMin: 16,
			},
		},
		"lowercase": {
			PasswordRequirements: api.PasswordRequirements{
				LowercaseMin: 1,
			},
			Settings: Settings{
				LowercaseMin: 1,
			},
		},
		"uppercase": {
			PasswordRequirements: api.PasswordRequirements{
				UppercaseMin: 1,
			},
			Settings: Settings{
				UppercaseMin: 1,
			},
		},
		"number": {
			PasswordRequirements: api.PasswordRequirements{
				NumberMin: 1,
			},
			Settings: Settings{
				NumberMin: 1,
			},
		},
		"symbol": {
			PasswordRequirements: api.PasswordRequirements{
				SymbolMin: 1,
			},
			Settings: Settings{
				SymbolMin: 1,
			},
		},
		"mixed": {
			PasswordRequirements: api.PasswordRequirements{
				LengthMin:    8,
				LowercaseMin: 1,
				UppercaseMin: 1,
			},
			Settings: Settings{
				LengthMin:    8,
				LowercaseMin: 1,
				UppercaseMin: 1,
			},
		},
	}

	for _, testCase := range cases {
		var actual Settings
		actual.SetFromAPI(&api.Settings{PasswordRequirements: testCase.PasswordRequirements})
		assert.DeepEqual(t, actual, testCase.Settings)
	}
}
