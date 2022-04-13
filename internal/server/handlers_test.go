package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestListProviders(t *testing.T) {
	s := setupServer(t)

	adminAccessKey := "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
	s.options = Options{AdminAccessKey: adminAccessKey}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	testProvider := &models.Provider{
		Name: "mokta",
	}

	err = data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.db)
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2)

	req, err := http.NewRequest(http.MethodGet, "/v1/providers", nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	apiProviders := make([]api.Provider, 0)
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders), 1)
	assert.Equal(t, apiProviders[0].Name, "mokta")
}

func TestDeleteProvider(t *testing.T) {
	s := setupServer(t)

	adminAccessKey := "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
	s.options = Options{AdminAccessKey: adminAccessKey}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	testProvider := &models.Provider{
		Name: "mokta",
	}

	err = data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/providers/%s", testProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestDeleteProvider_NoDeleteInternalProvider(t *testing.T) {
	s := setupServer(t)

	adminAccessKey := "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
	s.options = Options{AdminAccessKey: adminAccessKey}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/providers/%s", s.InternalProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

func TestCreateIdentity(t *testing.T) {
	s := setupServer(t)

	admin := &models.Identity{Name: "admin@example.com", Kind: models.UserKind}
	err := data.CreateIdentity(s.db, admin)
	assert.NilError(t, err)

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(s.db, adminGrant)
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", s.db)
	c.Set("identity", admin)

	handler := API{
		server: s,
	}

	t.Run("new unlinked user", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-create-identity@example.com",
			Kind: "user",
		}

		resp, err := handler.CreateIdentity(c, req)

		assert.NilError(t, err)
		assert.Equal(t, "test-create-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword == "")
	})

	t.Run("new infra user gets one time password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name:               "test-infra-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-infra-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})

	t.Run("existing unlinked user gets password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-link-identity@example.com",
			Kind: "user",
		}

		_, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		req = &api.CreateIdentityRequest{
			Name:               "test-link-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-link-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})
}

func TestDeleteIdentity(t *testing.T) {
	s := setupServer(t)

	adminAccessKey := "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
	s.options = Options{AdminAccessKey: adminAccessKey}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err = data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestDeleteIdentity_NoDeleteInternalIdentities(t *testing.T) {
	s := setupServer(t)

	adminAccessKey := "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb"
	s.options = Options{AdminAccessKey: adminAccessKey}

	err := s.importAccessKeys()
	assert.NilError(t, err)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	for name := range s.InternalIdentities {
		t.Run(name, func(t *testing.T) {
			route := fmt.Sprintf("/v1/identities/%s", s.InternalIdentities["admin"].ID)
			req, err := http.NewRequest(http.MethodDelete, route, nil)
			assert.NilError(t, err)

			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey))

			resp := httptest.NewRecorder()
			routes.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
		})
	}
}

func TestDeleteIdentity_NoDeleteSelf(t *testing.T) {
	s := setupServer(t)

	routes, err := s.GenerateRoutes(prometheus.NewRegistry())
	assert.NilError(t, err)

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err = data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	testAccessKey, err := data.CreateAccessKey(s.db, &models.AccessKey{Name: "test", IssuedFor: testUser.ID, ExpiresAt: time.Now().Add(time.Hour), ProviderID: s.InternalProvider.ID})
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}
