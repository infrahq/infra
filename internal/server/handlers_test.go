package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

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
