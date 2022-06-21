package models

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
)

// Internal Pagination Data
type Pagination struct {
	Page  int
	Limit int
}

func RequestToPagination(pr api.PaginationRequest) Pagination {
	if pr.Limit == 0 && pr.Page == 0 {
		return Pagination{} // temporary so pagination is disabled by default
	}
	page, limit := 1, 100

	if pr.Limit != 0 {
		limit = pr.Limit
	}

	if pr.Page != 0 {
		page = pr.Page
	}

	return Pagination{
		Page:  page,
		Limit: limit,
	}
}

func PaginationToResponse(c *gin.Context, p Pagination) api.PaginationResponse {

	if p == (Pagination{}) {
		return api.PaginationResponse{}
	}

	pr := api.PaginationResponse{
		Page:  p.Page,
		Limit: p.Limit,
	}

	SetURLs(&pr, c)

	return pr
}

func SetURLs(pr *api.PaginationResponse, c *gin.Context) {
	uri := *c.Request.URL
	uri.Host = c.Request.Host

	uri.Scheme = "https"
	if c.Request.TLS == nil {
		uri.Scheme = "http"
	}

	query := uri.Query()
	query.Set("limit", strconv.Itoa(pr.Limit))

	query.Set("page", strconv.Itoa(pr.Page)) // set self
	uri.RawQuery = query.Encode()
	pr.Self = uri.String()

	query.Set("page", strconv.Itoa(pr.Page+1)) // set next
	uri.RawQuery = query.Encode()
	pr.Next = uri.String()

	if pr.Page > 1 {
		query.Set("page", strconv.Itoa(pr.Page-1)) // set prev
		uri.RawQuery = query.Encode()
		pr.Prev = uri.String()
	}
}
