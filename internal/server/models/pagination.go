package models

import (
	"math"
)

// Internal Pagination Data
type Pagination struct {
	Page  int
	Limit int
	Sort  string

	TotalCount int
	TotalPages int
	Next       int
	Prev       int
}

func (p *Pagination) SetCount(count int64) {
	p.TotalPages = int(math.Ceil(float64(count) / float64(p.Limit)))
	p.TotalCount = int(count)

	if p.Next > p.TotalPages {
		p.Next = 0
	}
}

func (p *Pagination) SetDefaultSort(sort string) {
	if p.Sort == "" {
		p.Sort = sort
	}
}
