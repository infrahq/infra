package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ssoroka/slice"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
)

type Client struct {
	URL       string
	AccessKey string
	HTTP      http.Client
	// Headers are HTTP headers that will be added to every request made by the Client.
	Headers http.Header
}

// checkError checks the resp for an error code, and returns an api.Error with
// details about the error. Returns nil if the status code is 2xx.
//
// 3xx codes are considered an error because redirects should have already
// been followed before calling checkError.
func checkError(resp *http.Response, body []byte) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var apiError Error
	err := json.Unmarshal(body, &apiError)
	if err != nil {
		// Use the full body as the message if we fail to decode a response.
		apiError.Message = string(body)
		apiError.Code = int32(resp.StatusCode)
	}

	return apiError
}

// ErrorStatusCode returns the http status code from the error.
// Returns 0 if the error is nil, or if the error is not of type Error.
func ErrorStatusCode(err error) int32 {
	var apiError Error
	if errors.As(err, &apiError) {
		return apiError.Code
	}
	return 0
}

func get[Res any](client Client, path string) (*Res, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", client.URL, path), nil)
	if err != nil {
		return nil, err
	}

	addHeaders(req, client.Headers)
	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp, body)
	if err != nil {
		return nil, fmt.Errorf("GET %v failed: %w", path, err)
	}

	var res Res
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("parsing json response: %w. partial text: %q", err, partialText(body, 100))
	}

	return &res, nil
}

func list[Res any](client Client, path string, query map[string][]string) (*Res, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", client.URL, path), nil)
	if err != nil {
		return nil, err
	}

	addHeaders(req, client.Headers)
	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

	req.URL.RawQuery = url.Values(query).Encode()

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp, body)
	if err != nil {
		return nil, fmt.Errorf("GET %v failed: %w", path, err)
	}

	var res Res
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("parsing json response: %w. partial text: %q", err, partialText(body, 100))
	}

	return &res, nil
}

func request[Req, Res any](client Client, method string, path string, req *Req) (*Res, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", client.URL, path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	addHeaders(httpReq, client.Headers)
	httpReq.Header.Add("Authorization", "Bearer "+client.AccessKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp, body)
	if err != nil {
		return nil, fmt.Errorf("%s %v failed: %w", method, path, err)
	}

	var res Res
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("parsing json response: %w. partial text: %q", err, partialText(body, 100))
	}

	return &res, nil
}

func post[Req, Res any](client Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPost, path, req)
}

func put[Req, Res any](client Client, path string, req *Req) (res *Res, err error) {
	return request[Req, Res](client, http.MethodPut, path, req)
}

func delete(client Client, path string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s", client.URL, path), nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp, body)
	if err != nil {
		return fmt.Errorf("DELETE %v failed: %w", path, err)
	}

	return nil
}

func addHeaders(req *http.Request, headers http.Header) {
	req.Header.Set("Infra-Version", internal.FullVersion())
	for key, values := range headers {
		req.Header[key] = values
	}
}

func (c Client) ListUsers(req ListUsersRequest) (*ListResponse[User], error) {
	ids := slice.Map[uid.ID, string](req.IDs, func(id uid.ID) string {
		return id.String()
	})
	return list[ListResponse[User]](c, "/api/users", map[string][]string{"name": {req.Name}, "ids": ids})
}

func (c Client) GetUser(id uid.ID) (*User, error) {
	return get[User](c, fmt.Sprintf("/api/users/%s", id))
}

func (c Client) CreateUser(req *CreateUserRequest) (*CreateUserResponse, error) {
	return post[CreateUserRequest, CreateUserResponse](c, "/api/users", req)
}

func (c Client) UpdateUser(req *UpdateUserRequest) (*User, error) {
	return put[UpdateUserRequest, User](c, fmt.Sprintf("/api/users/%s", req.ID.String()), req)
}

func (c Client) DeleteUser(id uid.ID) error {
	return delete(c, fmt.Sprintf("/api/users/%s", id))
}

// Deprecated: use ListGrants
func (c Client) ListUserGrants(id uid.ID) (*ListResponse[Grant], error) {
	return list[ListResponse[Grant]](c, fmt.Sprintf("/api/users/%s/grants", id), nil)
}

func (c Client) ListUserGroups(id uid.ID) (*ListResponse[Group], error) {
	return list[ListResponse[Group]](c, fmt.Sprintf("/api/users/%s/groups", id), nil)
}

func (c Client) ListGroups(req ListGroupsRequest) (*ListResponse[Group], error) {
	return list[ListResponse[Group]](c, "/api/groups", map[string][]string{"name": {req.Name}})
}

func (c Client) GetGroup(id uid.ID) (*Group, error) {
	return get[Group](c, fmt.Sprintf("/api/groups/%s", id))
}

func (c Client) CreateGroup(req *CreateGroupRequest) (*Group, error) {
	return post[CreateGroupRequest, Group](c, "/api/groups", req)
}

// Deprecated: use ListGrants
func (c Client) ListGroupGrants(id uid.ID) (*ListResponse[Grant], error) {
	return list[ListResponse[Grant]](c, fmt.Sprintf("/api/groups/%s/grants", id), nil)
}

func (c Client) ListProviders(name string) (*ListResponse[Provider], error) {
	return list[ListResponse[Provider]](c, "/api/providers", map[string][]string{"name": {name}})
}

func (c Client) GetProvider(id uid.ID) (*Provider, error) {
	return get[Provider](c, fmt.Sprintf("/api/providers/%s", id))
}

func (c Client) CreateProvider(req *CreateProviderRequest) (*Provider, error) {
	return post[CreateProviderRequest, Provider](c, "/api/providers", req)
}

func (c Client) UpdateProvider(req UpdateProviderRequest) (*Provider, error) {
	return put[UpdateProviderRequest, Provider](c, fmt.Sprintf("/api/providers/%s", req.ID.String()), &req)
}

func (c Client) DeleteProvider(id uid.ID) error {
	return delete(c, fmt.Sprintf("/api/providers/%s", id))
}

func (c Client) ListGrants(req ListGrantsRequest) (*ListResponse[Grant], error) {
	return list[ListResponse[Grant]](c, "/api/grants", map[string][]string{
		"user":      {req.User.String()},
		"group":     {req.Group.String()},
		"resource":  {req.Resource},
		"privilege": {req.Privilege},
	})
}

func (c Client) CreateGrant(req *CreateGrantRequest) (*Grant, error) {
	return post[CreateGrantRequest, Grant](c, "/api/grants", req)
}

func (c Client) DeleteGrant(id uid.ID) error {
	return delete(c, fmt.Sprintf("/api/grants/%s", id))
}

func (c Client) ListDestinations(req ListDestinationsRequest) (*ListResponse[Destination], error) {
	return list[ListResponse[Destination]](c, "/api/destinations", map[string][]string{
		"name":      {req.Name},
		"unique_id": {req.UniqueID},
	})
}

func (c Client) CreateDestination(req *CreateDestinationRequest) (*Destination, error) {
	return post[CreateDestinationRequest, Destination](c, "/api/destinations", req)
}

func (c Client) UpdateDestination(req UpdateDestinationRequest) (*Destination, error) {
	return put[UpdateDestinationRequest, Destination](c, fmt.Sprintf("/api/destinations/%s", req.ID.String()), &req)
}

func (c Client) DeleteDestination(id uid.ID) error {
	return delete(c, fmt.Sprintf("/api/destinations/%s", id))
}

func (c Client) ListAccessKeys(req ListAccessKeysRequest) (*ListResponse[AccessKey], error) {
	return list[ListResponse[AccessKey]](c, "/api/access-keys", map[string][]string{
		"user_id": {req.UserID.String()},
		"name":    {req.Name},
	})
}

func (c Client) CreateAccessKey(req *CreateAccessKeyRequest) (*CreateAccessKeyResponse, error) {
	return post[CreateAccessKeyRequest, CreateAccessKeyResponse](c, "/api/access-keys", req)
}

func (c Client) DeleteAccessKey(id uid.ID) error {
	return delete(c, fmt.Sprintf("/api/access-keys/%s", id))
}

func (c Client) CreateToken() (*CreateTokenResponse, error) {
	return post[EmptyRequest, CreateTokenResponse](c, "/api/tokens", &EmptyRequest{})
}

func (c Client) Login(req *LoginRequest) (*LoginResponse, error) {
	return post[LoginRequest, LoginResponse](c, "/api/login", req)
}

func (c Client) Logout() error {
	_, err := post[EmptyRequest, EmptyResponse](c, "/api/logout", &EmptyRequest{})
	return err
}

func (c Client) SignupEnabled() (*SignupEnabledResponse, error) {
	return get[SignupEnabledResponse](c, "/api/signup")
}

func (c Client) Signup(req *SignupRequest) (*CreateAccessKeyResponse, error) {
	return post[SignupRequest, CreateAccessKeyResponse](c, "/api/signup", req)
}

func (c Client) GetVersion() (*Version, error) {
	return get[Version](c, "/api/version")
}

func partialText(body []byte, limit int) string {
	if len(body) <= limit {
		return string(body)
	}

	return string(body[:limit]) + "..."
}
