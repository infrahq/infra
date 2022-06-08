package api

type PaginationRequest struct {
	Page  int    `form:"page" validate:"min=0"`
	Limit int    `form:"limit" validate:"min=0,max=1000"`
	Sort  string `form:"sort" validate:"omitempty,oneof='name_ASC' 'name_DESC' 'id_ASC' 'id_DESC'"`
}

type PaginationResponse struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`

	Pages int `json:"pages"`
	Total int `json:"total"`
	Next  int `json:"next,omitempty"`
	Prev  int `json:"prev,omitempty"`
}
