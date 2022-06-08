package models

import (
	"math"
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
