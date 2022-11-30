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
	UserName  string `json:"userName"`
	Password  string `json:"password"`
	OrgName   string `json:"orgName"`
	Subdomain string `json:"subDomain"`
}

type reservedSubdomainData struct {
	Reject           []string `yaml:"reject"`
	RequiresApproval []string `yaml:"requires-approval"`
}

var ReservedSubdomains []string

//go:embed restricted_subdomains.yaml
var reservedSubdomainsYaml []byte

func init() {
	var reserved reservedSubdomainData
	if err := yaml.Unmarshal(reservedSubdomainsYaml, &reserved); err != nil {
		panic(err)
	}
	ReservedSubdomains = append(ReservedSubdomains, reserved.Reject...)
	ReservedSubdomains = append(ReservedSubdomains, reserved.RequiresApproval...)
}

func (r SignupOrg) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("userName", r.UserName),
		validate.Email("userName", r.UserName),
		validate.Required("password", r.Password),
		// check the admin user's password requirements against our basic password requirements
		validate.StringRule{
			Name:      "password",
			Value:     r.Password,
			MinLength: 8,
			MaxLength: 253,
		},
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
		validate.ReservedStrings("subDomain", r.Subdomain, ReservedSubdomains),
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
	Social *SocialSignup `json:"social"`
	Org    *SignupOrg    `json:"org"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "social", Value: r.Social},
			validate.Field{Name: "org", Value: r.Org},
		),
	}
}
