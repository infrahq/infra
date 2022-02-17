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
	Url       string
	AccessKey string
	Http      http.Client
}

func checkError(status int, body []byte) error {
	var apiError Error

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
	case http.StatusInternalServerError:
		return ErrInternal
	}

	return nil
}

func get[Res any](client Client, path string) (*Res, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", client.Url, path), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

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

	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

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

	httpReq.Header.Add("Authorization", "Bearer "+client.AccessKey)
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

func delete(client Client, path string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s", client.Url, path), nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

	resp, err := client.Http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) ListUsers(req ListUsersRequest) ([]User, error) {
	return list[User](c, "/v1/users", map[string]string{"email": req.Email, "provider_id": req.ProviderID.String()})
}

func (c Client) GetUser(id uid.ID) (*User, error) {
	return get[User](c, fmt.Sprintf("/v1/users/%s", id))
}

func (c Client) CreateUser(req *CreateUserRequest) (*User, error) {
	return post[CreateUserRequest, User](c, "/v1/users", req)
}

func (c Client) ListUserGrants(id uid.ID) ([]Grant, error) {
	return list[Grant](c, fmt.Sprintf("/v1/users/%s/grants", id), nil)
}

func (c Client) ListUserGroups(id uid.ID) ([]Group, error) {
	return list[Group](c, fmt.Sprintf("/v1/users/%s/groups", id), nil)
}

func (c Client) ListGroups(req ListGroupsRequest) ([]Group, error) {
	return list[Group](c, "/v1/groups", map[string]string{"name": req.Name, "provider_id": req.ProviderID.String()})
}

func (c Client) GetGroup(id uid.ID) (*Group, error) {
	return get[Group](c, fmt.Sprintf("/v1/groups/%s", id))
}

func (c Client) CreateGroup(req *CreateGroupRequest) (*Group, error) {
	return post[CreateGroupRequest, Group](c, "/v1/groups", req)
}

func (c Client) ListGroupGrants(id uid.ID) ([]Grant, error) {
	return list[Grant](c, fmt.Sprintf("/v1/groups/%s/grants", id), nil)
}

func (c Client) ListDestinations(req ListDestinationsRequest) ([]Destination, error) {
	return list[Destination](c, "/v1/destinations", map[string]string{"name": req.Name, "unique_id": req.UniqueID})
}

func (c Client) ListProviders(name string) ([]Provider, error) {
	return list[Provider](c, "/v1/providers", map[string]string{"name": name})
}

func (c Client) GetProvider(id uid.ID) (*Provider, error) {
	return get[Provider](c, fmt.Sprintf("/v1/providers/%s", id))
}

func (c Client) CreateProvider(req *CreateProviderRequest) (*Provider, error) {
	return post[CreateProviderRequest, Provider](c, "/v1/providers", req)
}

func (c Client) UpdateProvider(req UpdateProviderRequest) (*Provider, error) {
	return put[UpdateProviderRequest, Provider](c, fmt.Sprintf("/v1/providers/%s", req.ID.String()), &req)
}

func (c Client) DeleteProvider(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/providers/%s", id))
}

func (c Client) ListGrants(req ListGrantsRequest) ([]Grant, error) {
	return list[Grant](c, "/v1/grants", map[string]string{"resource": req.Resource, "identity": string(req.Identity), "privilege": req.Privilege})
}

func (c Client) CreateGrant(req *CreateGrantRequest) (*Grant, error) {
	return post[CreateGrantRequest, Grant](c, "/v1/grants", req)
}

func (c Client) DeleteGrant(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/grants/%s", id))
}

func (c Client) CreateDestination(req *CreateDestinationRequest) (*Destination, error) {
	return post[CreateDestinationRequest, Destination](c, "/v1/destinations", req)
}

func (c Client) UpdateDestination(req UpdateDestinationRequest) (*Destination, error) {
	return put[UpdateDestinationRequest, Destination](c, fmt.Sprintf("/v1/destinations/%s", req.ID.String()), &req)
}

func (c Client) DeleteDestination(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/destinations/%s", id))
}

func (c Client) Introspect() (*Introspect, error) {
	return get[Introspect](c, "/v1/introspect")
}

func (c Client) CreateAccessKey(req *CreateAccessKeyRequest) (*CreateAccessKeyResponse, error) {
	return post[CreateAccessKeyRequest, CreateAccessKeyResponse](c, "/v1/access-keys", req)
}

func (c Client) ListAccessKeys(req ListAccessKeysRequest) ([]AccessKey, error) {
	return list[AccessKey](c, "/v1/access-keys", map[string]string{"machineID": req.MachineID.String(), "name": req.Name})
}

func (c Client) DeleteAccessKey(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/access-keys/%s", id))
}

func (c Client) GetMachine(id uid.ID) (*Machine, error) {
	return get[Machine](c, fmt.Sprintf("/v1/machines/%s", id))
}

func (c Client) CreateMachine(req *CreateMachineRequest) (*Machine, error) {
	return post[CreateMachineRequest, Machine](c, "/v1/machines", req)
}

func (c Client) ListMachines(req ListMachinesRequest) ([]Machine, error) {
	return list[Machine](c, "/v1/machines", map[string]string{"name": req.Name})
}

func (c Client) DeleteMachine(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/machines/%s", id))
}

func (c Client) ListMachineGrants(id uid.ID) ([]Grant, error) {
	return list[Grant](c, fmt.Sprintf("/v1/machines/%s/grants", id), nil)
}

func (c Client) CreateToken(req *CreateTokenRequest) (*CreateTokenResponse, error) {
	return post[CreateTokenRequest, CreateTokenResponse](c, "/v1/tokens", req)
}

func (c Client) Login(req *LoginRequest) (*LoginResponse, error) {
	return post[LoginRequest, LoginResponse](c, "/v1/login", req)
}

func (c Client) Logout() error {
	_, err := post[EmptyRequest, EmptyResponse](c, "/v1/login", &EmptyRequest{})
	return err
}

func (c Client) GetVersion() (*Version, error) {
	return get[Version](c, "/v1/version")
}
