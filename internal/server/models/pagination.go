package models

import (
	"math"
	"strings"

	"github.com/infrahq/infra/api"
)

// Internal Pagination Data
type Pagination struct {
	Page  int
	Limit int
	Sort  string
	Total int
	Pages int
	Next  int
	Prev  int
}

// RequestToPagination takes a PaginationRequest from the api and converts to an internal Pagination model
func RequestToPagination(pr *api.PaginationRequest) Pagination {

	page, limit, sort := 1, 10, "id ASC"

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

	prevPage := page - 1
	if prevPage < 0 {
		prevPage = 0
	}

	return Pagination{
		Page:  page,
		Limit: limit,
		Sort:  sort,
		Prev:  prevPage,
		Next:  page + 1,
	}
}

// PaginationToResponse converts the internal Pagination model to a response sent to the user
func (p *Pagination) PaginationToResponse() api.PaginationResponse {
	if p.Prev < 0 {
		p.Prev = 0
	}

	return api.PaginationResponse{
		Page:  p.Page,
		Limit: p.Limit,
		Sort:  p.Sort,
		Prev:  p.Prev,
		Next:  p.Next,
		Pages: p.Pages,
		Total: p.Total,
	}
}

func (p *Pagination) SetCount(count int64) {
	p.Pages = int(math.Ceil(float64(count) / float64(p.Limit)))
	p.Total = int(count)

	if p.Next > p.Pages {
		p.Next = 0
	}
}

func (p *Pagination) SetDefaultSort(sort string) {
	if p.Sort == "" {
		p.Sort = sort
	}
}
