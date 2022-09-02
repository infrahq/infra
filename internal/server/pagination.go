package server

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
)

// PaginationFromRequest translates an api.PaginationRequest into the internal
// Pagination type.
func PaginationFromRequest(pr api.PaginationRequest) data.Pagination {
	page, limit := 1, 100

	if pr.Limit != 0 {
		limit = pr.Limit
	}

	if pr.Page != 0 {
		page = pr.Page
	}

	return data.Pagination{
		Page:  page,
		Limit: limit,
	}
}

// PaginationToResponse translates an internal Pagination type into the pagination
// response.
func PaginationToResponse(p data.Pagination) api.PaginationResponse {
	return api.PaginationResponse{
		Page:       p.Page,
		Limit:      p.Limit,
		TotalCount: p.TotalCount,
		TotalPages: p.TotalPages,
	}
}
