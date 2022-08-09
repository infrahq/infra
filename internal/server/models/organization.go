package models

import (
	"regexp"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	Model

	Name      string
	Domain    string
	CreatedBy uid.ID
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
	slug := domainNameReplacer.ReplaceAllStringFunc(o.Name, func(s string) string {
		if s == " " {
			return "-"
		}
		return ""
	})
	if len(slug) > 20 {
		slug = slug[:20]
	}
	o.Domain = slug + "-" + generate.MathRandom(5, generate.CharsetAlphaNumeric)
}
