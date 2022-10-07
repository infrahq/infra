package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type SCIMUserName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type SCIMUserEmail struct {
	Primary bool   `json:"primary"`
	Value   string `json:"value"`
}

func (r SCIMUserEmail) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("value", r.Value),
		validate.Email("value", r.Value),
	}
}

const UserSchema = "urn:ietf:params:scim:schemas:core:2.0:User"

type SCIMMetadata struct {
	ResourceType string `json:"resourceType"`
}

type SCIMUser struct {
	Schemas  []string        `json:"schemas"`
	ID       string          `json:"id"`
	UserName string          `json:"userName"`
	Name     SCIMUserName    `json:"name"`
	Emails   []SCIMUserEmail `json:"emails"`
	Active   bool            `json:"active"`
	Meta     SCIMMetadata    `json:"meta"`
}

type SCIMParametersRequest struct {
	// these pagination parameters must conform to the SCIM spec, rather than our standard pagination
	StartIndex int `form:"startIndex"`
	Count      int `form:"count"`
}

func (r SCIMParametersRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.IntRule{
			Name:  "startIndex",
			Value: r.StartIndex,
			Min:   validate.Int(0),
		},
		validate.IntRule{
			Name:  "count",
			Value: r.Count,
			Min:   validate.Int(0),
		},
	}
}

const ListResponseSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"

type ListProviderUsersResponse struct {
	Schemas      []string   `json:"schemas"`
	TotalResults int        `json:"totalResults"`
	Resources    []SCIMUser `json:"Resources"` // intentionally capitalized
	StartIndex   int        `json:"startIndex"`
	ItemsPerPage int        `json:"itemsPerPage"`
}

type SCIMUserProvisionRequest struct {
	Schemas  []string        `json:"schemas"`
	UserName string          `json:"userName"`
	Name     SCIMUserName    `json:"name"`
	Emails   []SCIMUserEmail `json:"emails"`
	Active   bool            `json:"active"`
}

func (r SCIMUserProvisionRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("schemas", r.Schemas),
	}
}

type SCIMUserUpdateRequest struct {
	ID       uid.ID          `uri:"id" json:"-"`
	Schemas  []string        `json:"schemas"`
	UserName string          `json:"userName"`
	Name     SCIMUserName    `json:"name"`
	Emails   []SCIMUserEmail `json:"emails"`
	Active   bool            `json:"active"`
}

func (r SCIMUserUpdateRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("schemas", r.Schemas),
		validate.Email("userName", r.UserName),
	}
}

const PatchOperationSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"

type SCIMPatchStatus struct {
	Active bool `json:"active"`
}

type SCIMPatchOperation struct {
	Op    string          `json:"op"`
	Value SCIMPatchStatus `json:"value"`
}

type SCIMUserPatchRequest struct {
	ID         uid.ID               `uri:"id" json:"-"`
	Schemas    []string             `json:"schemas"`
	Operations []SCIMPatchOperation `json:"Operations"` // json intentionally capitalized
}

func (r SCIMUserPatchRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("schemas", r.Schemas),
	}
}
