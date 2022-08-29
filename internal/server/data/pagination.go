package data

import "github.com/infrahq/infra/internal/server/data/querybuilder"

// Internal Pagination Data
type Pagination struct {
	Page       int
	Limit      int
	TotalCount int
}

func (p *Pagination) SetTotalCount(count int) {
	if p.Limit != 0 {
		p.TotalCount = count
	}
}

func (p *Pagination) PaginateQuery(query *querybuilder.Query) {
	if p.Page == 0 && p.Limit == 0 {
		return
	}

	offset := p.Limit * (p.Page - 1)
	query.B("LIMIT ? OFFSET ?", p.Limit, offset)
}
