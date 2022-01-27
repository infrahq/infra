package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/infrahq/infra/uid"
)

type Client struct {
	Url   string
	Token string
	Http  http.Client
}

func checkError(status int, body []byte) error {
	apiError := Error{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	}

	_ = json.Unmarshal(body, &apiError)

	switch apiError.Code {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusConflict:
		return ErrDuplicate
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrForbidden, apiError.Message)
	}

	return nil
}

func get[Res any](client Client, path string, query map[string]string) (*Res, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", client.Url, path), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+client.Token)

	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := client.Http.Do(req)
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

	var res Res
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func list[Res any](client Client, path string, query map[string]string) ([]Res, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", client.Url, path), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+client.Token)

	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := client.Http.Do(req)
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

	var res []Res
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func request[Req, Res any](client Client, method string, path string, req *Req) (*Res, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", client.Url, path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Add("Authorization", "Bearer "+client.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Http.Do(httpReq)
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

	var res Res
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func post[Req, Res any](client Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPost, path, req)
}

func put[Req, Res any](client Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPut, path, req)
}

func (c Client) ListUsers(email string) ([]User, error) {
	return list[User](c, "/v1/users", map[string]string{"email": email})
}

func (c Client) ListDestinations(nodeID string) ([]Destination, error) {
	return list[Destination](c, "/v1/destinations", map[string]string{"nodeID": nodeID})
}

func (c Client) ListProviders() ([]Provider, error) {
	return list[Provider](c, "/v1/providers", nil)
}

func (c Client) ListGrants(kind GrantKind, destinationID uid.ID) ([]Grant, error) {
	return list[Grant](c, "/v1/grants", map[string]string{"kind": string(kind), "destination_id": destinationID.String()})
}

func (c Client) CreateDestination(req *CreateDestinationRequest) (*Destination, error) {
	return post[CreateDestinationRequest, Destination](c, "/v1/destinations", req)
}

func (c Client) UpdateDestination(id uid.ID, req *UpdateDestinationRequest) (*Destination, error) {
	return put[UpdateDestinationRequest, Destination](c, fmt.Sprintf("/v1/destinations/%s", id.String()), req)
}

func (c Client) CreateToken(req *TokenRequest) (*Token, error) {
	return post[TokenRequest, Token](c, "/v1/tokens", req)
}

func (c Client) Login(req *LoginRequest) (*LoginResponse, error) {
	return post[LoginRequest, LoginResponse](c, "/v1/login", req)
}

func (c Client) Logout() error {
	_, err := post[EmptyRequest, EmptyResponse](c, "/v1/login", &EmptyRequest{})
	return err
}

func (c Client) GetVersion() (*Version, error) {
	return get[Version](c, "/v1/version", nil)
}
