package models

import (
	"math"

	"github.com/infrahq/infra/api"
)

// Internal Pagination Data
type Pagination struct {
	Page       int
	Limit      int
	TotalCount int
	TotalPages int
}

func RequestToPagination(pr api.PaginationRequest) Pagination {
	page, limit := 1, 100

	if pr.Limit != 0 {
		limit = pr.Limit
	}

	if pr.Page != 0 {
		page = pr.Page
	}

	return Pagination{
		Page:  page,
		Limit: limit,
	}
}

func PaginationToResponse(p Pagination) api.PaginationResponse {
	return api.PaginationResponse{
		Page:       p.Page,
		Limit:      p.Limit,
		TotalCount: p.TotalCount,
		TotalPages: p.TotalPages,
	}
}

func (p *Pagination) SetTotalCount(count int) {
	if p.Limit != 0 {
		p.TotalCount = count
		p.TotalPages = int(math.Ceil(float64(count) / float64(p.Limit)))
	}
}
