package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	Model

	Name      string
	Domain    string `gorm:"uniqueIndex:idx_organizations_domain,where:deleted_at is NULL"`
	CreatedBy uid.ID
}

func (o *Organization) ToAPI() *api.Organization {
	return &api.Organization{
		ID:     o.ID,
		Name:   o.Name,
		Domain: o.Domain,
	}
}

type OrganizationMember struct {
	// OrganizationID of the organization this entity belongs to.
	OrganizationID uid.ID
}

func (OrganizationMember) IsOrganizationMember() {}

func (o *OrganizationMember) SetOrganizationID(id uid.ID) {
	if o.OrganizationID == 0 {
		o.OrganizationID = id
	}
}
