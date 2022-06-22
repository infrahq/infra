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
	PageCount  int
}

func RequestToPagination(pr api.PaginationRequest) Pagination {
	if pr.Limit == 0 && pr.Page == 0 {
		return Pagination{} // temporary so pagination is disabled by default
	}
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
		Page:  p.Page,
		Limit: p.Limit,
	}
}

func (p *Pagination) SetTotalCount(count int) {
	p.TotalCount = count
	p.PageCount = int(math.Ceil(float64(count) / float64(p.Limit)))
}
