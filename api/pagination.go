package api

import (
	"math"
)

type PaginationRequest struct {
	Page  int    `form:"page"`
	Limit int    `form:"limit"`
	Sort  string `form:"sort" validate:"oneof='name ASC' 'name DESC' 'id ASC' 'id DESC' ''"`
}

type PaginationResponse struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`

	MaxPage int `json:"max_page"`
	Next    int `json:"next,omitempty"`
	Prev    int `json:"prev,omitempty"`
}

func GetMaxPage[T any](p *PaginationResponse, lr *ListResponse[T]) int {
	return int(math.Ceil(float64(lr.Count) / float64(p.Limit)))
}
