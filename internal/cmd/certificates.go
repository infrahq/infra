package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/infrahq/infra/pki"
	"github.com/spf13/cobra"
)

type KeyData struct {
	ClientKey *pki.KeyPair
	ServerKey *pki.KeyPair // private key section not used
}

func createCertificateCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "create (client|root)",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "client":
				return createClientCertificate()
			case "root":
				return createRootCertificate()
			default:
				return fmt.Errorf("invalid command %q", args[0])
			}
		},
	}
}

func trustCertificateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "trust",
		Args: cobra.ExactArgs(1),
	}

	opts := &struct {
		PEM string
		Key string
	}{}

	cmd.PersistentFlags().StringVar(&opts.PEM, "pem", "", "load the certificate to trust from a pem file")
	cmd.PersistentFlags().StringVar(&opts.Key, "key", "", "trust the server public key provided")

	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "instruct a server to trust a client",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Flags().Parse(args)
			if err != nil {
				return err
			}

			// Simply saves a client public cert in a place where the server will find it on startup.

			home, err := infraHomeDir()
			if err != nil {
				return err
			}

			path := filepath.Join(home, "keys", "trusted-client-keys")

			pems, raw, err := pki.ReadFromPEMFile(opts.PEM)
			if err != nil {
				return err
			}

			cert, err := x509.ParseCertificate(pems[0].Bytes)
			if err != nil {
				return err
			}

			err = os.MkdirAll(path, 0o700)
			if err != nil {
				return err
			}

			filename := filepath.Join(path, regexp.MustCompile(`[^\w\d]`).ReplaceAllString(cert.Subject.CommonName, "_")+".pem")

			err = os.WriteFile(filename, raw, 0o600)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.AddCommand(clientCmd)

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "instruct a client to trust a server",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Flags().Parse(args)
			if err != nil {
				return err
			}

			keydata, err := readLocalKeys()
			if err != nil {
				return err
			}

			pems, raw, err := pki.ReadFromPEMFile(opts.PEM)
			if err != nil {
				return err
			}

			cert, err := x509.ParseCertificate(pems[0].Bytes)
			if err != nil {
				return err
			}

			keydata.ServerKey = &pki.KeyPair{
				CertPEM:   raw,
				Cert:      cert,
				PublicKey: cert.PublicKey.(ed25519.PublicKey),
			}

			if err = writeLocalKeys(keydata); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.AddCommand(serverCmd)

	return cmd
}

func createClientCertificate() error {
	keydata, err := readLocalKeys()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if keydata == nil {
		keydata = &KeyData{}
	}

	if keydata.ClientKey != nil {
		path, _ := keysPath()
		return fmt.Errorf("file %q already contains client keys", path)
	}

	oneYear := 24 * time.Hour * 365 // TODO: auto-rotate client certs monthly
	keypair, err := pki.MakeUserCert("User ?", oneYear)
	if err != nil {
		return fmt.Errorf("creating user certificate: %w", err)
	}

	keydata.ClientKey = keypair

	if err := writeLocalKeys(keydata); err != nil {
		return err
	}

	return nil
}

func createRootCertificate() error {
	pub, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	serial, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("creating random serial: %w", err)
	}

	certTemplate := &x509.Certificate{
		SignatureAlgorithm: x509.PureEd25519,
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          pub,
		SerialNumber:       serial,
		Issuer:             pkix.Name{CommonName: "Root Infra CA"},
		Subject:            pkix.Name{CommonName: "Root Infra CA"},
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(7 * 24 * time.Hour), // temporary
		KeyUsage: x509.KeyUsageCertSign |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCRLSign |
			x509.KeyUsageKeyAgreement |
			x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,

		DNSNames:    []string{"localhost"}, // TODO: Support domain names for services?
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	// create client certificate from template and CA public key
	rawCert, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, pub, prv)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	fmt.Printf(`
server:
  certificates:
    initialRootCACert: %s
    initialRootCAPublicKey: %s
    initialRootCAPrivateKey: %s

These should be used for installing Infra, then discarded, as infra will manage key rotation itself.
You can trust the root certificate like so: 
    infra certificates trust server --key %s
`,
		base64.StdEncoding.EncodeToString(rawCert),
		base64.StdEncoding.EncodeToString(pub),
		base64.StdEncoding.EncodeToString(prv),
		base64.StdEncoding.EncodeToString(pub),
	)

	return nil
}

// readLocalKeys reads the client's local keys
func readLocalKeys() (*KeyData, error) {
	path, err := keysPath()
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filepath.Join(path, "keys.json"))
	if os.IsNotExist(err) {
		return nil, ErrConfigNotFound
	}

	keydata := &KeyData{}
	err = json.Unmarshal(contents, keydata)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return keydata, nil
}

func keysPath() (string, error) {
	infraDir, err := infraHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(infraDir, "keys", "client")
	_ = os.MkdirAll(path, 0o700)

	return path, nil
}

func writeLocalKeys(keydata *KeyData) error {
	path, err := keysPath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(keydata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(path, "keys.json"), data, 0o600)
	if err != nil {
		return err
	}

	pem, err := pki.MarshalPrivateKey(keydata.ClientKey.PrivateKey)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(path, "key.pem"), pem, 0o600)
	if err != nil {
		return err
	}

	data = keydata.ClientKey.CertPEM
	if len(keydata.ClientKey.SignedCertPEM) > 0 {
		data = keydata.ClientKey.SignedCertPEM
	}

	err = ioutil.WriteFile(filepath.Join(path, "cert.pem"), data, 0o600)
	if err != nil {
		return err
	}

	if keydata.ServerKey != nil {
		err = ioutil.WriteFile(filepath.Join(path, "server.crt"), keydata.ServerKey.CertPEM, 0o600)
		if err != nil {
			return err
		}
	}

	return nil
}
