package pki

import (
	"os"
	"testing"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/secrets"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	fp := secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{
		Path: os.TempDir(),
	})

	kp := secrets.NewNativeSecretProvider(fp)

	key, err := kp.GenerateDataKey("")
	require.NoError(t, err)

	models.SymmetricKey = key

	return db
}

func TestCertificateStorage(t *testing.T) {
	cfg := NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: 2,
	}

	db := setupDB(t)

	p, err := NewNativeCertificateProvider(db, cfg)
	require.NoError(t, err)

	err = p.CreateCA()
	require.NoError(t, err)

	activeCAs := p.ActiveCAs()
	require.Len(t, activeCAs, 2)

	// reload
	p, err = NewNativeCertificateProvider(db, cfg)
	require.NoError(t, err)

	reloadedActiveCAs := p.ActiveCAs()
	require.Len(t, reloadedActiveCAs, 2)

	require.Equal(t, activeCAs, reloadedActiveCAs)
}

func TestTLSCertificates(t *testing.T) {
	cfg := NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: 2,
	}
	p, err := NewNativeCertificateProvider(setupDB(t), cfg)
	require.NoError(t, err)

	err = p.CreateCA()
	require.NoError(t, err)

	activeCAs := p.ActiveCAs()
	require.Len(t, activeCAs, 2)

	certs, err := p.TLSCertificates()
	require.NoError(t, err)
	require.Len(t, certs, 2)
}
