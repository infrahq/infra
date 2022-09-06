package server

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestMetrics(t *testing.T) {
	run := func(db *data.DB, s string) []byte {
		patchProductVersion(t, "9.9.9")
		registry := setupMetrics(db)

		tempfile, err := ioutil.TempFile(t.TempDir(), t.Name())
		assert.NilError(t, err)

		err = prometheus.WriteToTextfile(tempfile.Name(), registry)
		assert.NilError(t, err)

		bts, err := ioutil.ReadFile(tempfile.Name())
		assert.NilError(t, err)
		assert.Assert(t, len(bts) > 0)

		re := regexp.MustCompile(s)
		return bytes.Join(re.FindAll(bts, -1), []byte("\n"))
	}

	t.Run("build info", func(t *testing.T) {
		db := setupDB(t)
		actual := run(db, `build_info({.*})? \d+`)
		expected := `build_info{branch="main",commit="",date="",version="9.9.9"} 1`
		assert.Equal(t, string(actual), expected)
	})

	t.Run("infra users", func(t *testing.T) {
		db := setupDB(t)

		assert.NilError(t, data.CreateIdentity(db, &models.Identity{Name: "1"}))
		assert.NilError(t, data.CreateIdentity(db, &models.Identity{Name: "2"}))
		assert.NilError(t, data.CreateIdentity(db, &models.Identity{Name: "3"}))

		actual := run(db, `infra_users({.*})? \d+`)
		golden.Assert(t, string(actual), t.Name())
	})

	t.Run("infra groups", func(t *testing.T) {
		db := setupDB(t)

		assert.NilError(t, data.CreateGroup(db, &models.Group{Name: "heroes"}))
		assert.NilError(t, data.CreateGroup(db, &models.Group{Name: "villains"}))

		actual := run(db, `infra_groups({.*})? \d+`)
		golden.Assert(t, string(actual), t.Name())
	})

	t.Run("infra grants", func(t *testing.T) {
		db := setupDB(t)

		actual := run(db, `infra_grants({.*})? \d+`)
		golden.Assert(t, string(actual), t.Name())
	})

	t.Run("infra providers", func(t *testing.T) {
		db := setupDB(t)

		assert.NilError(t, data.CreateProvider(db, &models.Provider{Name: "oidc", Kind: "oidc"}))
		assert.NilError(t, data.CreateProvider(db, &models.Provider{Name: "okta", Kind: "okta"}))
		assert.NilError(t, data.CreateProvider(db, &models.Provider{Name: "okta2", Kind: "okta"}))
		assert.NilError(t, data.CreateProvider(db, &models.Provider{Name: "azure", Kind: "azure"}))
		assert.NilError(t, data.CreateProvider(db, &models.Provider{Name: "google", Kind: "google"}))

		actual := run(db, `infra_providers({.*})? \d+`)
		golden.Assert(t, string(actual), t.Name())
	})

	t.Run("infra destinations", func(t *testing.T) {
		db := setupDB(t)

		assert.NilError(t, data.CreateDestination(db, &models.Destination{Name: "1", UniqueID: "1", LastSeenAt: time.Now()}))
		assert.NilError(t, data.CreateDestination(db, &models.Destination{Name: "2", UniqueID: "2", Version: "", LastSeenAt: time.Now().Add(-10 * time.Minute)}))
		assert.NilError(t, data.CreateDestination(db, &models.Destination{Name: "3", UniqueID: "3", Version: "0.1.0", LastSeenAt: time.Now()}))
		assert.NilError(t, data.CreateDestination(db, &models.Destination{Name: "4", UniqueID: "4", Version: "0.1.0"}))
		assert.NilError(t, data.CreateDestination(db, &models.Destination{Name: "5", UniqueID: "5", Version: "0.1.0"}))

		actual := run(db, `infra_destinations({.*})? \d+`)
		golden.Assert(t, string(actual), t.Name())
	})
}
