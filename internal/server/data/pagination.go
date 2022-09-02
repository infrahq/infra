package data

import "math"

// Internal Pagination Data
type Pagination struct {
	Page       int
	Limit      int
	TotalCount int
	TotalPages int
}

func (p *Pagination) SetTotalCount(count int) {
	if p.Limit != 0 {
		p.TotalCount = count
		p.TotalPages = int(math.Ceil(float64(count) / float64(p.Limit)))
	}
}
