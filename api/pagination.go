package api

import "github.com/infrahq/infra/internal/validate"

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
	Page       int `json:"page,omitempty"`
	Limit      int `json:"limit,omitempty"`
	TotalPages int `json:"totalPages,omitempty"`
	TotalCount int `json:"totalCount,omitempty"`
}
