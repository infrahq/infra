package models

import "github.com/infrahq/infra/api"

// Internal Pagination Data
type Pagination struct {
	Page  int
	Limit int
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

func PaginationToResponse(pr Pagination) api.PaginationResponse {
	return api.PaginationResponse{
		Page:  pr.Page,
		Limit: pr.Limit,
	}
}
