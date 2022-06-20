package models

import (
	"fmt"
	"regexp"
	"strings"

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

	uri := *c.Request.URL
	uri.Host = c.Request.Host
	uri.Scheme = "https" // TODO: get proper scheme

	pr := api.PaginationResponse{
		Page:    p.Page,
		Limit:   p.Limit,
		Current: uri.String(),
	}

	if strings.Contains(uri.RawQuery, "page=") {
		regex := regexp.MustCompile("page=[0-9]*")
		uri.RawQuery = regex.ReplaceAllString(uri.RawQuery, "page=%d")
	} else {
		uri.RawQuery += "&page=%d"
	}

	pr.Next = fmt.Sprintf(uri.String(), pr.Page+1)

	if pr.Page > 0 {
		pr.Prev = fmt.Sprintf(uri.String(), pr.Page-1)
	}

	return pr
}
