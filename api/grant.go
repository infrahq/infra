package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id" note:"ID of grant created" example:"3w9XyTrkzk"`

	Created   Time   `json:"created"`
	CreatedBy uid.ID `json:"createdBy" note:"id of the user that created the grant"`
	Updated   Time   `json:"updated"`

	User      uid.ID `json:"user,omitempty" note:"UserID for a user being granted access" example:"6hNnjfjVcc"`
	Group     uid.ID `json:"group,omitempty" note:"GroupID for a group being granted access" example:"3zMaadcd2U"`
	Privilege string `json:"privilege" note:"a role or permission" example:"admin"`

	DestinationName     string `json:"destinationName"`
	DestinationResource string `json:"destinationResource,omitempty"`
}

type CreateGrantResponse struct {
	*Grant     `json:",inline"`
	WasCreated bool `json:"-" note:"Indicates that grant was successfully created, false it already existed beforehand" example:"true"`
}

func (r *CreateGrantResponse) StatusCode() int {
	if !r.WasCreated {
		return http.StatusOK
	}
	return http.StatusCreated
}

type ListGrantsRequest struct {
	User                uid.ID `form:"user" note:"ID of user granted access" example:"6TjWTAgYYu"`
	Group               uid.ID `form:"group" note:"ID of group granted access" example:"6k3Eqcqu6B"`
	DestinationName     string `form:"destinationName"`
	DestinationResource string `form:"destinationResource"`
	Privilege           string `form:"privilege" example:"view" note:"a role or permission"`
	ShowInherited       bool   `form:"showInherited" note:"if true, this field includes grants that the user inherits through groups" example:"true"`
	ShowSystem          bool   `form:"showSystem" note:"if true, this shows the connector and other internal grants" example:"false"`
	BlockingRequest
	PaginationRequest
}

func (r ListGrantsRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.MutuallyExclusive(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "group", Value: r.Group},
		),
		validate.ValidatorFunc(func() *validate.Failure {
			if r.ShowInherited && r.User == 0 {
				return validate.Fail("showInherited", "requires a user ID")
			}
			return nil
		}),
		validate.ValidatorFunc(r.validateLastUpdateIndex),
	}
}

func (r ListGrantsRequest) validateLastUpdateIndex() *validate.Failure {
	if r.LastUpdateIndex == 0 {
		return nil
	}

	// At least one of the supported query parameters must be set, and no other
	// query parameters can be set
	switch {
	case r.DestinationName != "":
		if fields := r.fieldsWithValues("destinationName", "lastUpdateIndex"); len(fields) > 0 {
			return validate.Fail("lastUpdateIndex",
				fmt.Sprintf("can not be used with %v parameter(s)", strings.Join(fields, ",")))
		}

	default:
		return validate.Fail("lastUpdateIndex", "requires a supported filter")
	}
	return nil
}

// TODO: completeness test
func (r ListGrantsRequest) fieldsWithValues(ignored ...string) []string {
	var result []string
	add := func(v string) {
		result = append(result, v)
	}
	ignore := func(v string) bool {
		for _, item := range ignored {
			if item == v {
				return true
			}
		}
		return false
	}
	if r.User != 0 && !ignore("user") {
		add("user")
	}
	if r.Group != 0 && !ignore("group") {
		add("group")
	}
	if r.DestinationName != "" && !ignore("destinationName") {
		add("destinationName")
	}
	if r.DestinationResource != "" && !ignore("destinationResource") {
		add("destinationResource")
	}
	if r.Privilege != "" && !ignore("privilege") {
		add("privilege")
	}
	if r.ShowSystem && !ignore("showSystem") {
		add("showSystem")
	}
	if r.ShowInherited && !ignore("showInherited") {
		add("showInherited")
	}
	if r.LastUpdateIndex != 0 && !ignore("lastUpdateIndex") {
		add("lastUpdateIndex")
	}
	if r.Page != 0 && !ignore("page") {
		add("page")
	}
	if r.Limit != 0 && !ignore("limit") {
		add("limit")
	}
	return result
}

func (r ListGrantsRequest) SetPage(page int) Paginatable {
	r.PaginationRequest.Page = page
	return r
}

// GrantRequest defines a grant request which can be used for creating or deleting grants
type GrantRequest struct {
	User                uid.ID `json:"user" note:"ID of the user granted access" example:"6kdoMDd6PA"`
	Group               uid.ID `json:"group" note:"ID of the group granted access" example:"6Ti2p7r1h7"`
	UserName            string `json:"userName" note:"Name of the user granted access" example:"admin@example.com"`
	GroupName           string `json:"groupName" note:"Name of the group granted access" example:"dev"`
	Privilege           string `json:"privilege" example:"view" note:"a role or permission"`
	DestinationName     string `json:"destinationName"`
	DestinationResource string `json:"destinationResource"`
}

func (r GrantRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "userName", Value: r.UserName},
			validate.Field{Name: "group", Value: r.Group},
			validate.Field{Name: "groupName", Value: r.GroupName},
		),
		validate.Required("privilege", r.Privilege),
		validate.Required("destinationName", r.DestinationName),
	}
}

type UpdateGrantsRequest struct {
	GrantsToAdd    []GrantRequest `json:"grantsToAdd" note:"List of grant objects. See POST api/grants for more"`
	GrantsToRemove []GrantRequest `json:"grantsToRemove" note:"List of grant objects. See POST api/grants for more"`
}

func ParseResourceURN(urn string) (name, resource string) {
	name, resource, _ = strings.Cut(urn, ".")
	return name, resource
}

func FormatResourceURN(name, resource string) string {
	urn := name
	if resource != "" {
		urn = fmt.Sprintf("%s.%s", name, resource)
	}

	return urn
}
