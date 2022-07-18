package cmd

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

// listAll is a helper function that handles pagination and calls the given list request function.
// listItems is the corresponding function in the API client that handles the Request "req".
// handleError is a function that handles the error returned by the API client.
func listAll[Item any, Req api.Paginatable](listItems func(Req) (*api.ListResponse[Item], error), req Req) ([]Item, error) {

	logging.Debugf("call server: page 1")

	req, ok := req.SetPage(1).(Req)
	if !ok {
		panic("SetPage returned a different request type than expected")
	}

	res, err := listItems(req)
	if err != nil {
		return nil, err
	}
	users := make([]Item, 0, res.TotalCount)
	users = append(users, res.Items...)

	for page := 2; page <= res.TotalPages; page++ {
		req, ok := req.SetPage(page).(Req)
		if !ok {
			panic("SetPage returned a different request type than expected")
		}

		logging.Debugf("call server: page %d", page)
		res, err = listItems(req)
		if err != nil {
			return nil, err
		}
		users = append(users, res.Items...)
	}

	return users, nil
}
