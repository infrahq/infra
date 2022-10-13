package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   Time   `json:"created"`
	CreatedBy uid.ID `json:"created_by" note:"id of the user that created the grant"`
	Updated   Time   `json:"updated"`

	User      uid.ID `json:"user,omitempty"`
	Group     uid.ID `json:"group,omitempty"`
	Privilege string `json:"privilege" note:"a role or permission"`
	Resource  string `json:"resource" note:"a resource name in Infra's Universal Resource Notation"`
}

type CreateGrantResponse struct {
	*Grant     `json:",inline"`
	WasCreated bool `json:"wasCreated"`
}

func (r *CreateGrantResponse) StatusCode() int {
	if !r.WasCreated {
		return http.StatusOK
	}
	return http.StatusCreated
}

type ListGrantsRequest struct {
	User            uid.ID `form:"user"`
	Group           uid.ID `form:"group"`
	Resource        string `form:"resource" example:"production.namespace"`
	Destination     string `form:"destination" example:"production"`
	Privilege       string `form:"privilege" example:"view"`
	ShowInherited   bool   `form:"showInherited" note:"if true, this field includes grants that the user inherits through groups"`
	ShowSystem      bool   `form:"showSystem" note:"if true, this shows the connector and other internal grants"`
	LastUpdateIndex int64  `form:"lastUpdateIndex" note:"set this to the value of the Last-Update-Index response header to block until the list results have changed"`
	PaginationRequest
}

func (r ListGrantsRequest) ValidationRules() []validate.ValidationRule {
	destNameRule := validateDestinationName(r.Destination)
	destNameRule.Name = "destination"

	return []validate.ValidationRule{
		validate.MutuallyExclusive(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "group", Value: r.Group},
		),
		validate.MutuallyExclusive(
			validate.Field{Name: "resource", Value: r.Resource},
			validate.Field{Name: "destination", Value: r.Destination},
		),
		destNameRule,
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
	case r.Destination != "":
		// TODO: require limit=-1
		if fields := r.fieldsWithValues("destination", "lastUpdateIndex"); len(fields) > 0 {
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
	if r.Resource != "" && !ignore("resource") {
		add("resource")
	}
	if r.Destination != "" && !ignore("destination") {
		add("destination")
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

type CreateGrantRequest struct {
	User      uid.ID `json:"user"`
	Group     uid.ID `json:"group"`
	Privilege string `json:"privilege" example:"view" note:"a role or permission"`
	Resource  string `json:"resource" example:"production" note:"a resource name in Infra's Universal Resource Notation"`
}

func (r CreateGrantRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "group", Value: r.Group},
		),
		validate.Required("privilege", r.Privilege),
		validate.Required("resource", r.Resource),
	}
}
