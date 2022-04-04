package pki

import (
	"crypto/x509"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	db, err := data.NewDB(driver)
	assert.NilError(t, err)

	fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	kp := secrets.NewNativeSecretProvider(fp)

	key, err := kp.GenerateDataKey("")
	assert.NilError(t, err)

	models.SymmetricKey = key

	return db
}

func TestCertificateStorage(t *testing.T) {
	cfg := NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: 2,
	}

	db := setupDB(t)

	p, err := NewNativeCertificateProvider(db, cfg)
	assert.NilError(t, err)

	err = p.CreateCA()
	assert.NilError(t, err)

	activeCAs := p.ActiveCAs()
	assert.Assert(t, is.Len(activeCAs, 2))

	// reload
	p, err = NewNativeCertificateProvider(db, cfg)
	assert.NilError(t, err)

	reloadedActiveCAs := p.ActiveCAs()
	assert.Assert(t, is.Len(reloadedActiveCAs, 2))

	assert.DeepEqual(t, activeCAs, reloadedActiveCAs, cmpX509Certificate)
}

// cmpX509Certificate compares two x509.Certificate using the Equal method.
// go-cmp is supposed to use an Equal method automatically, but I guess the
// pointer receiver and pointer arg to Equal are preventing that.
var cmpX509Certificate = cmp.Comparer(func(x, y x509.Certificate) bool {
	return x.Equal(&y)
})

func TestTLSCertificates(t *testing.T) {
	cfg := NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: 2,
	}
	p, err := NewNativeCertificateProvider(setupDB(t), cfg)
	assert.NilError(t, err)

	err = p.CreateCA()
	assert.NilError(t, err)

	activeCAs := p.ActiveCAs()
	assert.Assert(t, is.Len(activeCAs, 2))

	certs, err := p.TLSCertificates()
	assert.NilError(t, err)
	assert.Assert(t, is.Len(certs, 2))
}
