package api

import (
	"math"
)

type PaginationRequest struct {
	Page  int    `form:"page" validate:"min=0"`
	Limit int    `form:"limit" validate:"min=0,max=250"`
	Sort  string `form:"sort" validate:"oneof='name_ASC' 'name_DESC' 'id_ASC' 'id_DESC' ''"`
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
