package api

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"github.com/infrahq/infra/internal/validate"
)

type SignupResponse struct {
	User         *User         `json:"user"`
	Organization *Organization `json:"organization"`
}

type SignupOrg struct {
	Name      string `json:"name"`
	Subdomain string `json:"subDomain"`
}

type reservedSubdomainData struct {
	Reject           []string `yaml:"reject"`
	RequiresApproval []string `yaml:"requires-approval"`
}

var reservedSubdomains []string

//go:embed restricted_subdomains.yaml
var reservedSubdomainsYaml []byte

func init() {
	var reserved reservedSubdomainData
	if err := yaml.Unmarshal(reservedSubdomainsYaml, &reserved); err != nil {
		panic(err)
	}
	reservedSubdomains = append(reservedSubdomains, reserved.Reject...)
	reservedSubdomains = append(reservedSubdomains, reserved.RequiresApproval...)
}

func (r SignupOrg) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("subDomain", r.Subdomain),
		validate.StringRule{
			Name:      "subDomain",
			Value:     r.Subdomain,
			MinLength: 6,
			MaxLength: 63,
			CharacterRanges: []validate.CharRange{
				validate.AlphabetLower,
				validate.AlphabetUpper,
				validate.Numbers,
				validate.Dash,
			},
			FirstCharacterRange: []validate.CharRange{
				validate.AlphabetLower,
				validate.AlphabetUpper,
				validate.Numbers,
			},
		},
		validate.ReservedStrings("subDomain", r.Subdomain, reservedSubdomains),
	}
}

type SignupRequest struct {
	Name     string    `json:"name"`
	Password string    `json:"password"`
	Org      SignupOrg `json:"org"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Email("name", r.Name),
		validate.Required("password", r.Password),
		// check the admin user's password requirements against our basic password requirements
		validate.StringRule{
			Name:      "password",
			Value:     r.Password,
			MinLength: 8,
			MaxLength: 253,
		},
	}
}
