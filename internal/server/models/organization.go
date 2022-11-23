package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	Model

	Name           string
	Domain         string
	AllowedDomains CommaSeparatedStrings // the email domains that are allowed to login to this org

	CreatedBy uid.ID
}

func (o *Organization) ToAPI() *api.Organization {
	return &api.Organization{
		ID:             o.ID,
		Name:           o.Name,
		Created:        api.Time(o.CreatedAt),
		Updated:        api.Time(o.UpdatedAt),
		Domain:         o.Domain,
		AllowedDomains: o.AllowedDomains,
	}
}

type OrganizationMember struct {
	// OrganizationID of the organization this entity belongs to.
	OrganizationID uid.ID
}

func (OrganizationMember) IsOrganizationMember() {}

type OrganizationIDSource interface {
	OrganizationID() uid.ID
}

func (o *OrganizationMember) SetOrganizationID(source OrganizationIDSource) {
	if o.OrganizationID == 0 {
		o.OrganizationID = source.OrganizationID()
	}
	if o.OrganizationID == 0 {
		panic("OrganizationID was not set")
	}
}
