package api

import "github.com/infrahq/infra/internal/validate"

type Paginatable interface {
	SetPage(page int) Paginatable
}

type PaginationRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
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
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
	TotalCount int `json:"totalCount"`
}
