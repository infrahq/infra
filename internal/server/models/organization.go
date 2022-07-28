package models

import (
	"regexp"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	Model

	Name      string `gorm:"uniqueIndex:idx_organizations_name,where:deleted_at is NULL"`
	Domain    string `gorm:"uniqueIndex:idx_org_domain,where:deleted_at is NULL"`
	CreatedBy uid.ID

	Identities []Identity `gorm:"many2many:identities_organizations"`
}

func (o *Organization) ToAPI() *api.Organization {
	return &api.Organization{
		ID:     o.ID,
		Name:   o.Name,
		Domain: o.Domain,
	}
}

var domainNameReplacer = regexp.MustCompile(`[^\da-zA-Z-]`)

func (o *Organization) SetDefaultDomain() {
	if len(o.Domain) > 0 {
		return
	}
	o.Domain = domainNameReplacer.ReplaceAllStringFunc(o.Name, func(s string) string {
		if s == " " {
			return "-"
		}
		return ""
	}) + "-" + generate.MathRandom(5, generate.CharsetAlphaNumeric)
}
