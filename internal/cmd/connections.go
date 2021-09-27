package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func canReachInternet() (bool, error) {
	req, err := http.NewRequest("GET", "https://google.com", nil)
	if err != nil {
		return false, err
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	return true, nil
}

func canConnectToEndpoint(endpoint string) (bool, error) {
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // ok for testing connections
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
	req, err := http.NewRequest("GET", endpoint, nil)
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
	_, err = client.Do(req)
	if err != nil {
		return false, err
	}

	return true, nil
}

func canGetEngineStatus(row statusRow) (bool, error) {
	endpoint := row.Endpoint
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	req, err := http.NewRequest("GET", endpoint+"/healthz", nil)
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
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, errors.New("unexpected response code " + resp.Status)
	}

	return true, nil
}
