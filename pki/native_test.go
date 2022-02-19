package pki

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCertificateDiskStorage(t *testing.T) {
	s, err := os.MkdirTemp(os.TempDir(), "certs")
	require.NoError(t, err)

	cfg := NativeCertificateProviderConfig{
		StoragePath:                   s,
		FullKeyRotationDurationInDays: 2,
	}
	p, err := NewNativeCertificateProvider(cfg)
	require.NoError(t, err)

	err = p.CreateCA()
	require.NoError(t, err)

	activeCAs := p.ActiveCAs()
	require.Len(t, activeCAs, 2)

	// reload
	p, err = NewNativeCertificateProvider(cfg)
	require.NoError(t, err)

	reloadedActiveCAs := p.ActiveCAs()
	require.Len(t, reloadedActiveCAs, 2)

	require.Equal(t, activeCAs, reloadedActiveCAs)
}

func TestTLSCertificates(t *testing.T) {
	s, err := os.MkdirTemp(os.TempDir(), "certs")
	require.NoError(t, err)

	cfg := NativeCertificateProviderConfig{
		StoragePath:                   s,
		FullKeyRotationDurationInDays: 2,
	}
	p, err := NewNativeCertificateProvider(cfg)
	require.NoError(t, err)

	err = p.CreateCA()
	require.NoError(t, err)

	activeCAs := p.ActiveCAs()
	require.Len(t, activeCAs, 2)

	certs, err := p.TLSCertificates()
	require.NoError(t, err)
	require.Len(t, certs, 2)
}
