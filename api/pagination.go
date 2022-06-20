package api

type PaginationRequest struct {
	Page  int `form:"page" validate:"min=0"`
	Limit int `form:"limit" validate:"min=0,max=1000"`
}

type PaginationResponse struct {
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`

	Next    string `json:"next,omitempty"`
	Current string `json:"current,omitempty"`
	Prev    string `json:"previous,omitempty"`
	//TODO: add some indication of number of records/pages or if a next page exists
}
