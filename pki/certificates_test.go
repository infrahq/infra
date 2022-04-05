package pki

import (
	"crypto/ed25519"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/infrahq/infra/testutil/docker"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestMain(m *testing.M) {
	defer func() {
		if r := recover(); r != nil {
			teardown()
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	flag.Parse()
	setup()

	result := m.Run()

	teardown()
	// nolint
	os.Exit(result)
}

var containerIDs []string

func setup() {
	if testing.Short() {
		return
	}
}

func teardown() {
	if testing.Short() {
		return
	}

	for _, containerID := range containerIDs {
		docker.KillContainer(containerID)
	}
}

func eachProvider(t *testing.T, eachFunc func(t *testing.T, p CertificateProvider)) {
	providers := map[string]CertificateProvider{}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "certificates")
	assert.NilError(t, err)

	defer os.RemoveAll(tmpDir)

	db := setupDB(t)

	p, err := NewNativeCertificateProvider(db, NativeCertificateProviderConfig{})
	assert.NilError(t, err)

	providers["native"] = p

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			eachFunc(t, provider)
		})
	}
}

func TestCertificatesImplementations(t *testing.T) {
	eachProvider(t, func(t *testing.T, p CertificateProvider) {
		err := p.CreateCA()
		assert.NilError(t, err)

		certs := p.ActiveCAs()
		// should have two keys now
		require.InDelta(t, 182*day, time.Until(certs[0].NotAfter), float64(1*day))
		require.InDelta(t, 365*day, time.Until(certs[1].NotAfter), float64(1*day))

		err = p.RotateCA()
		assert.NilError(t, err)

		certs = p.ActiveCAs()

		for i, cert := range certs {
			cert := cert
			t.Run("check cert "+strconv.Itoa(i), func(t *testing.T) {
				assert.Assert(t, cert.IsCA)
				assert.Assert(t, cert.NotBefore.Before(time.Now()))
				require.InDelta(t, 365*day, time.Until(cert.NotAfter), float64(1*day))
				assert.Equal(t, "Root Infra CA", cert.Subject.CommonName)
			})
		}

		t.Run("signing Cert Signing Requests", func(t *testing.T) {
			cert, err := generateClientCertificate("Connector")
			assert.NilError(t, err)

			csr := x509.CertificateRequest{
				PublicKeyAlgorithm: cert.PublicKeyAlgorithm,
				PublicKey:          cert.PublicKey,

				Signature:          cert.Signature,
				SignatureAlgorithm: cert.SignatureAlgorithm,
				Subject:            cert.Subject,
				Extensions:         cert.Extensions,
			}

			pemBytes, err := p.SignCertificate(csr)
			assert.NilError(t, err)

			block, rest := pem.Decode(pemBytes)
			assert.Assert(t, is.Len(rest, 0))

			cert, err = x509.ParseCertificate(block.Bytes)
			assert.NilError(t, err)

			parent := p.ActiveCAs()[1]

			err = parent.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature)
			assert.NilError(t, err)
		})
	})
}

func init() {
	// only used in tests
	randReader = rand.New(rand.NewSource(0)) //nolint:gosec
}

func generateClientCertificate(subject string) (*x509.Certificate, error) {
	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return nil, fmt.Errorf("generating keys: %w", err)
	}

	kp := KeyPair{
		PublicKey:  pub,
		PrivateKey: prv,
	}

	cert, _, err := createClientCertSignedBy(kp, kp, subject, 1*time.Minute)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

func createClientCertSignedBy(signer, signee KeyPair, subject string, lifetime time.Duration) (*x509.Certificate, []byte, error) {
	sig := ed25519.Sign(signer.PrivateKey, signee.PublicKey)
	if !ed25519.Verify(signer.PublicKey, signee.PublicKey, sig) {
		return nil, nil, errors.New("self-signed certificate doesn't match signature")
	}

	certTemplate := x509.Certificate{
		Signature:          sig,
		SignatureAlgorithm: x509.PureEd25519,
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          signee.PublicKey,
		SerialNumber:       big.NewInt(rand.Int63()), //nolint:gosec
		Subject:            pkix.Name{CommonName: subject},
		NotBefore:          time.Now().Add(-5 * time.Minute),
		NotAfter:           time.Now().Add(lifetime),
		KeyUsage:           x509.KeyUsageDataEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	// create client certificate from template and CA public key
	rawCert, err := x509.CreateCertificate(randReader, &certTemplate, &certTemplate, signee.PublicKey, signee.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing self-created certificate: %w", err)
	}

	return cert, rawCert, nil
}
