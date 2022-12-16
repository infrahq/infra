package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestDestinationCredentials(t *testing.T) {
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	db := srv.db
	routes := srv.GenerateRoutes()
	destName := "foo"
	wg := sync.WaitGroup{}
	wg.Add(1)

	admin := createAdmin(t, db)
	key := createAccessKeyForUser(t, db, admin)

	err := data.CreateDestination(db, &models.Destination{
		OrganizationMember: models.OrganizationMember{OrganizationID: admin.OrganizationID},
		Name:               destName,
		Kind:               models.DestinationKindKubernetes,
	})
	assert.NilError(t, err)

	err = data.CreateGrant(db, &models.Grant{
		OrganizationMember: models.OrganizationMember{OrganizationID: admin.OrganizationID},
		Subject:            uid.NewIdentityPolymorphicID(admin.ID),
		Privilege:          "admin",
		Resource:           destName,
	})
	assert.NilError(t, err)

	t.Run("CreateDestinationCredential", func(t *testing.T) {
		t.Parallel()

		wg.Wait()
		resp, status, err := httpReq[*api.CreateDestinationCredential, api.DestinationCredential](t, routes, http.MethodPost, "/api/credentials", &api.CreateDestinationCredential{
			Destination: destName,
		}, key)

		assert.NilError(t, err)
		assert.Equal(t, status, 201)

		assert.Equal(t, resp.BearerToken, "abc.123")
		assert.Equal(t, resp.CredentialExpiresAt, api.Time(time.Date(2053, 12, 30, 0, 0, 0, 0, time.UTC)))
	})

	t.Run("ListDestinationCredentials", func(t *testing.T) {
		t.Parallel()
		go wg.Done()
		resp, status, err := httpReq[*api.ListDestinationCredential, api.ListDestinationCredentialResponse](t, routes, http.MethodGet, "/api/credentials", &api.ListDestinationCredential{
			Destination: destName,
		}, key)
		assert.NilError(t, err)
		assert.Equal(t, status, 200)

		assert.Equal(t, len(resp.Items), 1)

		t.Run("AnswerDestinationCredential", func(t *testing.T) {
			r := resp.Items[0]

			_, status, err := httpReq[*api.AnswerDestinationCredential, api.EmptyResponse](t, routes, http.MethodPut, "/api/credentials", &api.AnswerDestinationCredential{
				ID:                  r.ID,
				OrganizationID:      r.OrganizationID,
				BearerToken:         "abc.123",
				CredentialExpiresAt: api.Time(time.Date(2053, 12, 30, 0, 0, 0, 0, time.UTC)),
			}, adminAccessKey(srv))
			assert.NilError(t, err)
			assert.Equal(t, status, 200)

		})
	})
}

func httpReq[Req any, Resp any](t *testing.T, routes Routes, method, path string, req Req, accessKey string) (*Resp, int, error) {
	body := jsonBody(t, req)
	r := httptest.NewRequest(method, path, body)
	r.Header.Set("Authorization", "Bearer "+accessKey)
	r.Header.Set("Infra-Version", apiVersionLatest)

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, r)

	respObj := new(Resp)
	err := json.Unmarshal(resp.Body.Bytes(), respObj)
	status := resp.Result().StatusCode
	if status < 300 {
		assert.NilError(t, err)
	}

	return respObj, status, nil

}
