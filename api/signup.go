package api

import "github.com/infrahq/infra/internal/validate"

type SignupResponse struct {
	User         *User         `json:"user"`
	Organization *Organization `json:"organization"`
}

type SignupOrg struct {
	Name      string `json:"name"`
	Subdomain string `json:"subDomain"`
}

var reservedSubDomains = []string{
	"infra", "infrahq", "auth", "authz", "authn",
	"api", "www", "ftp", "ssh", "info", "help", "about",
	"grants", "connector", "login", "signup",
	"system", "admin", "email", "bastion",
}

func (r SignupOrg) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("subDomain", r.Subdomain),
		validate.ReservedStrings("subDomain", r.Subdomain, reservedSubDomains),
		validate.StringRule{
			Name:      "subDomain",
			Value:     r.Subdomain,
			MinLength: 3,
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
	}
}
