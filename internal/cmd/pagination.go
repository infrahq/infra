package cmd

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

func listAll[Item any, Req api.Paginatable](client *api.Client, req Req, listItems func(api.Client, Req) (*api.ListResponse[Item], error), handleError func(err error) error) ([]Item, error) {

	logging.Debugf("call server: page 1")
	req = req.SetPage(1).(Req)
	res, err := listItems(*client, req)
	if err != nil {
		return nil, handleError(err)
	}
	users := make([]Item, 0, res.TotalCount)
	users = append(users, res.Items...)

	// first page done in first request
	for page := 2; page <= res.TotalPages; page++ {
		logging.Debugf("call server: page %d", page)
		req = req.SetPage(page).(Req)
		res, err = listItems(*client, req)
		if err != nil {
			return nil, handleError(err)
		}
		users = append(users, res.Items...)
	}

	return users, nil
}
