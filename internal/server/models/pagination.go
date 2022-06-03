package models

import "github.com/infrahq/infra/api"

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

func RequestToPagination(pr *api.PaginationRequest) Pagination {

	var page, limit int
	var sort string

	if pr == nil || pr.Page == 0 {
		page = 1
	} else {
		page = pr.Page
	}

	if pr == nil || pr.Limit == 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	} else {
		limit = pr.Limit
	}

	if pr == nil || pr.Sort == "" {
		sort = "name ASC"
	} else {
		sort = pr.Sort
	}

	return Pagination{
		Page:  page,
		Limit: limit,
		Sort:  sort,
		Prev:  page - 1,
		Next:  page + 1,
	}
}

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
