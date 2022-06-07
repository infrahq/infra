package models

import (
	"strings"

	"github.com/infrahq/infra/api"
)

// Internal Pagination Data
type Pagination struct {
	Page    int
	Limit   int
	Sort    string
	MaxPage int
	Next    int
	Prev    int
}

// function to convert API --> this object
// method to convert Pagination to response

// RequestToPagination takes a PaginationRequest from the api and converts to an internal Pagination model
func RequestToPagination(pr *api.PaginationRequest) Pagination {

	page, limit, sort := 1, 10, "name ASC"

	if pr != nil {
		if pr.Page != 0 {
			page = pr.Page
		}

		if pr.Limit != 0 {
			limit = pr.Limit
		}

		if pr.Sort != "" {
			sort = strings.ReplaceAll(pr.Sort, "_", " ")
		}
	}
	return Pagination{
		Page:  page,
		Limit: limit,
		Sort:  sort,
		Prev:  page - 1,
		Next:  page + 1,
	}
}

// PaginationToResponse converts the internal Pagination model to a response sent to the user
func (pg *Pagination) PaginationToResponse() api.PaginationResponse {
	if pg.Prev < 0 {
		pg.Prev = 0
	}

	// if pg.Next > pg.MaxPage {
	// 	pg.Next = 0
	// }
	//TODO: fix MaxPage — decide on design

	return api.PaginationResponse{
		Page:    pg.Page,
		Limit:   pg.Limit,
		Sort:    pg.Sort,
		Prev:    pg.Prev,
		Next:    pg.Next,
		MaxPage: pg.MaxPage,
	}
}
