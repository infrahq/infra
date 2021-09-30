package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func canReachInternet() (bool, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://google.com", nil)
	if err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return true, nil
}

func canConnectToEndpoint(endpoint string) (bool, error) {
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}

	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				//nolint:gosec // ok for testing connections
				InsecureSkipVerify: true,
			},
		},
	}

	_, err = client.Do(req)
	if err != nil {
		return false, err
	}

	return true, nil
}

func canConnectToTLSEndpoint(row statusRow) (bool, error) {
	endpoint := row.Endpoint
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}

	caCertPool := x509.NewCertPool()

	if len(row.CertificateAuthorityData) > 0 {
		fmt.Println("🐞🪲🐛 adding CA")
		caCertPool.AppendCertsFromPEM([]byte(row.CertificateAuthorityData))
	}

	// this should use the same TLS configuration as the rest of the app
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    caCertPool,
			},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return true, nil
}

func canGetEngineStatus(row statusRow) (bool, error) {
	endpoint := row.Endpoint
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint+"/healthz", nil)
	if err != nil {
		return false, err
	}

	caCertPool := x509.NewCertPool()
	if len(row.CertificateAuthorityData) > 0 {
		caCertPool.AppendCertsFromPEM(row.CertificateAuthorityData)
	}

	// this should use the same TLS configuration as the rest of the app
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				RootCAs:    caCertPool,
			},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return false, errors.New("unexpected response code " + res.Status)
	}

	return true, nil
}
