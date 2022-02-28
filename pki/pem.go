package pki

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

func ReadFromPEMFile(file string) (pems []*pem.Block, pemBytes []byte, err error) {
	b, err := readFile(file)
	if err != nil {
		return nil, nil, err
	}

	for {
		block, rest := pem.Decode(b)
		if block == nil && bytes.Equal(rest, b) {
			return nil, nil, fmt.Errorf("%q contains no pem data", file)
		}

		if block != nil {
			pems = append(pems, block)
		}

		if len(rest) == 0 {
			break
		}
	}

	return pems, b, nil
}

func MarshalPrivateKey(key ed25519.PrivateKey) ([]byte, error) {
	marshalledPrvKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshalling private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrvKey,
	})

	return keyPEM, nil
}

func readFile(file string) ([]byte, error) {
	// nicer errors from os.Stat. it'll be an errors.Is(err, os.ErrNotExist) if it doesn't exist.
	if _, err := os.Stat(file); err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", file, err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", file, err)
	}

	return b, nil
}
