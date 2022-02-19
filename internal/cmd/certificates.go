package cmd

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	}{}

	cmd.PersistentFlags().StringVar(&opts.PEM, "pem", "", "load the certificate to trust from a pem file")

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
				CertRaw:   raw,
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
	keydata, err := readLocalKeys()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return err
	}

	if keydata == nil {
		keydata = &KeyData{}
	}

	// if keydata.ServerKey != nil {
	// 	path, _ := keysPath()
	// 	return fmt.Errorf("file %q already contains root server keys", path)
	// }

	dir, err := infraHomeDir()
	if err != nil {
		return err
	}

	storagePath := filepath.Join(dir, "keys")

	cp, err := pki.NewNativeCertificateProvider(pki.NativeCertificateProviderConfig{
		StoragePath:                   storagePath,
		FullKeyRotationDurationInDays: 365,
	})
	if err != nil {
		return err
	}

	if len(cp.ActiveCAs()) > 0 {
		return fmt.Errorf("root certificates already exist at " + storagePath)
	}

	if err := cp.CreateCA(); err != nil {
		return err
	}

	fmt.Printf(`
CA keys written to %s/root.crt and %s/root.key. These should be used for installing Infra, then discarded.
You can trust the root certificate like so: 
		infra certificates trust server --pem %s/root.crt
`, storagePath, storagePath, storagePath)

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

	data = keydata.ClientKey.CertRaw
	if len(keydata.ClientKey.SignedCertRaw) > 0 {
		data = keydata.ClientKey.SignedCertRaw
	}

	err = ioutil.WriteFile(filepath.Join(path, "cert.pem"), data, 0o600)
	if err != nil {
		return err
	}

	if keydata.ServerKey != nil {
		err = ioutil.WriteFile(filepath.Join(path, "server.crt"), keydata.ServerKey.CertRaw, 0o600)
		if err != nil {
			return err
		}
	}

	return nil
}
