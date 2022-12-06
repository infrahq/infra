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

type SignupUser struct {
	UserName string `json:"username"`
	Password string `json:"password"`
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

func (r SignupUser) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("username", r.UserName),
		validate.Email("username", r.UserName),
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

type SocialSignup struct {
	Code        string `json:"code"`
	RedirectURL string `json:"redirectURL"`
}

func (r SocialSignup) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("code", r.Code),
		validate.Required("redirectURL", r.RedirectURL),
	}
}

type SignupRequest struct {
	Social    *SocialSignup `json:"social"`
	User      *SignupUser   `json:"user"`
	OrgName   string        `json:"orgName"`
	Subdomain string        `json:"subDomain"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "social", Value: r.Social},
			validate.Field{Name: "user", Value: r.User},
		),

		validate.Required("orgName", r.OrgName),
		validate.Required("subDomain", r.Subdomain),
		validate.StringRule{
			Name:      "subDomain",
			Value:     r.Subdomain,
			MinLength: 4,
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
