package api

import "github.com/infrahq/infra/internal/validate"

type Paginatable interface {
	SetPage(page int) Paginatable
}

type PaginationRequest struct {
	Page  int `form:"page" note:"Page number to retrieve" example:"1"`
	Limit int `form:"limit" note:"Number of objects to retrieve per page (up to 1000)" example:"100"`
}

func (p PaginationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.IntRule{
			Name:  "page",
			Value: p.Page,
			Min:   validate.Int(0),
		},
		validate.IntRule{
			Name:  "limit",
			Value: p.Limit,
			Min:   validate.Int(0),
			Max:   validate.Int(1000),
		},
	}
}

type PaginationResponse struct {
	Page       int `json:"page" note:"Page number retrieved" example:"1"`
	Limit      int `json:"limit" note:"Number of objects per page" example:"100"`
	TotalPages int `json:"totalPages" note:"Total number of pages" example:"5"`
	TotalCount int `json:"totalCount" note:"Total number of objects" example:"485"`
}
