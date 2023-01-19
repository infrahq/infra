package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/claims"
)

type authenticator struct {
	mu          sync.Mutex
	key         *jose.JSONWebKey
	lastChecked time.Time

	client          httpClient
	baseURL         string
	serverAccessKey string
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func newAuthenticator(options Options) *authenticator {
	transport := httpTransportFromOptions(options.Server)
	return &authenticator{
		client:          &http.Client{Transport: transport},
		baseURL:         options.Server.URL.String(),
		serverAccessKey: options.Server.AccessKey.String(),
	}
}

var JWKCacheRefresh = 5 * time.Minute

func (j *authenticator) Authenticate(req *http.Request) (claims.Custom, error) {
	c := claims.Custom{}
	authHeader := req.Header.Get("Authorization")

	raw := strings.TrimPrefix(authHeader, "Bearer ")
	if raw == "" {
		return c, fmt.Errorf("no bearer token found")
	}

	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		return c, fmt.Errorf("invalid JWT signature: %w", err)
	}

	key, err := j.getJWK()
	if err != nil {
		return c, fmt.Errorf("get JWK from server: %w", err)
	}

	var allClaims struct {
		jwt.Claims
		claims.Custom
	}
	if err := tok.Claims(key, &allClaims); err != nil {
		return c, fmt.Errorf("invalid token claims: %w", err)
	}

	err = allClaims.Claims.Validate(jwt.Expected{Time: time.Now().UTC()})
	switch {
	case errors.Is(err, jwt.ErrExpired):
		return c, err
	case err != nil:
		return c, fmt.Errorf("invalid JWT %w", err)
	}

	if allClaims.Custom.Name == "" {
		return c, fmt.Errorf("no username in JWT claims")
	}

	return allClaims.Custom, nil
}

func (j *authenticator) getJWK() (*jose.JSONWebKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.lastChecked.IsZero() && time.Now().Before(j.lastChecked.Add(JWKCacheRefresh)) {
		return j.key, nil
	}

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, fmt.Sprintf("%s/.well-known/jwks.json", j.baseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+j.serverAccessKey)

	res, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected response: %v ", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	if len(response.Keys) < 1 {
		return nil, errors.New("no jwks provided by infra")
	}

	j.lastChecked = time.Now().UTC()
	j.key = &response.Keys[0]

	return &response.Keys[0], nil
}
