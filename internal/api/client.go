package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Http http.Client
}

func checkError(status int, body []byte) error {
	apiError := Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	}

	_ = json.Unmarshal(body, &apiError)

	// TODO: finish these
	switch apiError.Code {
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrForbidden, apiError.Message)
	}

	return nil
}

func get[Res any](client http.Client, path string, query map[string]string) (res *Res, err error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range query {
		req.URL.Query().Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func list[Res any](client http.Client, path string, query map[string]string) ([]Res, error) {
	res, err := get[[]Res](client, path, query)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

func request[Req, Res any](client http.Client, method string, path string, req *Req) (res *Res, err error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func post[Req, Res any](client http.Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPost, path, req)
}

func put[Req, Res any](client http.Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPut, path, req)
}

func (c *Client) ListUsers(email string) ([]User, error) {
	return list[User](c.Http, "/v1/users", map[string]string{"email": email})
}

func (c *Client) ListDestinations(nodeID string) ([]Destination, error) {
	return list[Destination](c.Http, "/v1/destinations", map[string]string{"nodeID": nodeID})
}

func (c *Client) ListProviders() ([]Provider, error) {
	return list[Provider](c.Http, "/v1/providers", nil)
}

func (c *Client) ListGrants(kind GrantKind, destinationID string) ([]Grant, error) {
	return list[Grant](c.Http, "/v1/grants", map[string]string{"kind": string(kind), "destination_id": destinationID})
}

func (c *Client) CreateDestination(req *DestinationRequest) (*Destination, error) {
	return post[DestinationRequest, Destination](c.Http, "/v1/destinations", req)
}

func (c *Client) UpdateDestination(id string, req *DestinationRequest) (*Destination, error) {
	return put[DestinationRequest, Destination](c.Http, fmt.Sprintf("/v1/destinations/%s", id), req)
}

func (c *Client) CreateToken(req *TokenRequest) (*Token, error) {
	return post[TokenRequest, Token](c.Http, "/v1/tokens", req)
}

func (c *Client) Login(req *LoginRequest) (*LoginResponse, error) {
	return post[LoginRequest, LoginResponse](c.Http, "/v1/login", req)
}

func (c *Client) Logout() error {
	_, err := post[EmptyRequest, EmptyResponse](c.Http, "/v1/login", &EmptyRequest{})
	return err
}

func (c *Client) GetVersion() (*Version, error) {
	return get[Version](c.Http, "/v1/version", nil)
}
