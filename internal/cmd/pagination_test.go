package cmd

import (
	"fmt"
	"testing"

	"github.com/infrahq/infra/api" // nolint
	"gotest.tools/v3/assert"
)

func TestListAll(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		users, err := listAll(mockListUsers, api.ListUsersRequest{Name: "empty"})
		assert.NilError(t, err)

		assert.DeepEqual(t, users, []api.User{})
	})

	t.Run("one", func(t *testing.T) {
		users, err := listAll(mockListUsers, api.ListUsersRequest{Name: "one"})
		assert.NilError(t, err)

		assert.DeepEqual(t, users, []api.User{
			{Name: "jonathan@test.com"},
		})
	})

	t.Run("two", func(t *testing.T) {
		users, err := listAll(mockListUsers, api.ListUsersRequest{Name: "two"})
		assert.NilError(t, err)

		assert.DeepEqual(t, users, []api.User{{Name: "1@test.com"}, {Name: "2@test.com"}})

	})

	t.Run("five", func(t *testing.T) {
		users, err := listAll(mockListUsers, api.ListUsersRequest{Name: "five"})
		assert.NilError(t, err)

		assert.DeepEqual(t, users, []api.User{
			{Name: "1@test.com"}, {Name: "1@test.org"}, {Name: "2@test.com"}, {Name: "2@test.org"}, {Name: "3@test.com"},
			{Name: "3@test.org"}, {Name: "4@test.com"}, {Name: "4@test.org"}, {Name: "5@test.com"}, {Name: "5@test.org"},
		})
	})

	t.Run("error", func(t *testing.T) {
		_, err := listAll(mockListUsers, api.ListUsersRequest{Name: "error"})
		assert.Error(t, err, "default error")
	})

}

func mockListUsers(req api.ListUsersRequest) (*api.ListResponse[api.User], error) {
	switch req.Name {
	case "empty":
		return &api.ListResponse[api.User]{
			Items:              []api.User{},
			PaginationResponse: api.PaginationResponse{TotalPages: 0, TotalCount: 0, Page: req.Page},
		}, nil
	case "one":
		return &api.ListResponse[api.User]{
			Items:              []api.User{{Name: "jonathan@test.com"}},
			PaginationResponse: api.PaginationResponse{TotalPages: 1, TotalCount: 1, Page: req.Page},
		}, nil
	case "two":
		return &api.ListResponse[api.User]{
			Items:              []api.User{{Name: fmt.Sprintf("%d@test.com", req.Page)}},
			PaginationResponse: api.PaginationResponse{TotalPages: 2, TotalCount: 2, Page: req.Page},
		}, nil
	case "five":
		return &api.ListResponse[api.User]{
			Items:              []api.User{{Name: fmt.Sprintf("%d@test.com", req.Page)}, {Name: fmt.Sprintf("%d@test.org", req.Page)}},
			PaginationResponse: api.PaginationResponse{TotalPages: 5, TotalCount: 5, Page: req.Page},
		}, nil
	case "403":
		return nil, api.Error{Code: 403}
	default:
		return nil, Error{Message: "default error"}
	}
}
