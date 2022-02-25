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

	err := json.Unmarshal(body, &apiError)
	if err != nil {
		apiError.Message = string(body)
		apiError.Code = int32(status)
	}

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
		return fmt.Errorf("%w: %s", ErrBadRequest, apiError.Message)
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
		return nil, fmt.Errorf("GET %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return nil, fmt.Errorf("GET %q responded %d: %w", path, resp.StatusCode, err)
	}

	var res Res
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("parsing json response: %w. partial text: %q", err, partialText(body, 100))
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
		return nil, fmt.Errorf("GET %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return nil, fmt.Errorf("GET %q responded %d: %w", path, resp.StatusCode, err)
	}

	var res []Res
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("parsing json response: %w. partial text: %q", err, partialText(body, 100))
	}

	return res, nil
}

func request[Req, Res any](client Client, method string, path string, req *Req) (*Res, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", client.Url, path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Add("Authorization", "Bearer "+client.AccessKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return nil, fmt.Errorf("%s %q responded %d: %w", method, path, resp.StatusCode, err)
	}

	var res Res
	err = json.Unmarshal(body, &res)
	if err != nil {
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
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s%s", client.Url, path), nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+client.AccessKey)

	resp, err := client.Http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %q: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	err = checkError(resp.StatusCode, body)
	if err != nil {
		return fmt.Errorf("DELETE %q responded %d: %w", path, resp.StatusCode, err)
	}

	return nil
}

// @title        Infra API
// @version      1.0
// @BasePath     /v1
// @host         api.infrahq.com
// @securityDefinitions.apiKey AccessKey
// @in header
// @name Authorization

// @tag.name Users
// @tag.description Manage Users

// ListUsers     godoc
// @Summary   List all users
// @Tags      Users
// @Security  AccessKey
// @Produce   json
// @Param     email        query    string  false  "email to filter by"
// @Param     provider_id  query    string  false  "provider id to filter by"
// @Success   200          {array}  User
// @Router    /users [get]
func (c Client) ListUsers(req ListUsersRequest) ([]User, error) {
	return list[User](c, "/v1/users", map[string]string{"email": req.Email, "provider_id": req.ProviderID.String()})
}

// GetUser       godoc
// @Summary   Retrieve a user
// @Tags      Users
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200  {object}  User
// @Router    /users/{id} [get]
func (c Client) GetUser(id uid.ID) (*User, error) {
	return get[User](c, fmt.Sprintf("/v1/users/%s", id))
}

// CreateUser    godoc
// @Summary   Create a user
// @Tags      Users
// @Security  AccessKey
// @Param     body  body  CreateUserRequest  true  "Parameters"
// @Produce   json
// @Success   201  {object}  User
// @Router    /users [post]
func (c Client) CreateUser(req *CreateUserRequest) (*User, error) {
	return post[CreateUserRequest, User](c, "/v1/users", req)
}

// ListUserGrants    godoc
// @Summary   List a user's grants
// @Tags      Users
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200  {object}  Grant
// @Router    /users/{id}/grants [get]
func (c Client) ListUserGrants(id uid.ID) ([]Grant, error) {
	return list[Grant](c, fmt.Sprintf("/v1/users/%s/grants", id), nil)
}

// ListUserGroups    godoc
// @Summary   List a user's groups
// @Tags      Users
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200  {array}  Group
// @Router    /users/{id}/groups [get]
func (c Client) ListUserGroups(id uid.ID) ([]Group, error) {
	return list[Group](c, fmt.Sprintf("/v1/users/%s/groups", id), nil)
}

// @tag.name Groups
// @tag.description Manage Groups

// ListGroups        godoc
// @Summary   List all groups
// @Tags      Groups
// @Security  AccessKey
// @Param     name         query  string  false  "group name to filter by"
// @Param     provider_id  query  string  false  "provider id to filter by"
// @Produce   json
// @Success   200  {array}  Group
// @Router    /groups [get]
func (c Client) ListGroups(req ListGroupsRequest) ([]Group, error) {
	return list[Group](c, "/v1/groups", map[string]string{"name": req.Name, "provider_id": req.ProviderID.String()})
}

// GetGroup          godoc
// @Summary   Retrieve a group
// @Tags      Groups
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200  {object}  Group
// @Router    /groups/{id} [get]
func (c Client) GetGroup(id uid.ID) (*Group, error) {
	return get[Group](c, fmt.Sprintf("/v1/groups/%s", id))
}

// CreateGroup       godoc
// @Summary   Create a group
// @Tags      Groups
// @Security  AccessKey
// @Param     body  body  CreateGroupRequest  true  "Parameters"
// @Accept    json
// @Produce   json
// @Success   201  {object}  Group
// @Router    /groups [post]
func (c Client) CreateGroup(req *CreateGroupRequest) (*Group, error) {
	return post[CreateGroupRequest, Group](c, "/v1/groups", req)
}

// ListGroupGrants   godoc
// @Summary   List grants for a group
// @Tags      Groups
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200  {array}  Group
// @Router    /users/{id}/groups [get]
func (c Client) ListGroupGrants(id uid.ID) ([]Grant, error) {
	return list[Grant](c, fmt.Sprintf("/v1/groups/%s/grants", id), nil)
}

// @tag.name Providers
// @tag.description Manage Providers

// ListProviders     godoc
// @Summary  List all providers
// @Tags     Providers
// @Param    name  query  string  false  "name to filter by"
// @Produce  json
// @Success  200  {array}  Provider
// @Router   /providers [get]
func (c Client) ListProviders(name string) ([]Provider, error) {
	return list[Provider](c, "/v1/providers", map[string]string{"name": name})
}

// GetProvider       godoc
// @Summary  Retrieve a provider
// @Tags     Providers
// @Param    id  path  string  true  "Unique ID"
// @Produce  json
// @Success  200  {object}  Provider
// @Router   /providers/{id} [get]
func (c Client) GetProvider(id uid.ID) (*Provider, error) {
	return get[Provider](c, fmt.Sprintf("/v1/providers/%s", id))
}

// CreateProvider    godoc
// @Summary   Connect a provider
// @Tags      Providers
// @Security  AccessKey
// @Param     body  body  CreateProviderRequest  true  "Parameters"
// @Accept    json
// @Produce   json
// @Success   201  {array}  Provider
// @Router    /providers [post]
func (c Client) CreateProvider(req *CreateProviderRequest) (*Provider, error) {
	return post[CreateProviderRequest, Provider](c, "/v1/providers", req)
}

// UpdateProvider    godoc
// @Summary   Update a provider
// @Tags      Providers
// @Security  AccessKey
// @Param     id    path  string                 true  "Unique ID"
// @Param     body  body  UpdateProviderRequest  true  "Parameters"
// @Produce   json
// @Success   200  {object}  Provider
// @Router    /providers/{id} [put]
func (c Client) UpdateProvider(req UpdateProviderRequest) (*Provider, error) {
	return put[UpdateProviderRequest, Provider](c, fmt.Sprintf("/v1/providers/%s", req.ID.String()), &req)
}

// DeleteProvider    godoc
// @Summary   Delete a provider
// @Tags      Providers
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200
// @Router    /providers/{id} [delete]
func (c Client) DeleteProvider(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/providers/%s", id))
}

// @tag.name Grants
// @tag.description Manage Grants

// ListGrants        godoc
// @Summary   List all grants
// @Tags      Grants
// @Security  AccessKey
// @Param     resource   query  string  false  "resource to filter by"
// @Param     identity   query  string  false  "identity id to filter by, prefixed by u:, g: or m:"
// @Param     privilege  query  string  false  "privilege to filter by"
// @Produce   json
// @Success   200  {array}  Grant
// @Router    /grants [get]
func (c Client) ListGrants(req ListGrantsRequest) ([]Grant, error) {
	return list[Grant](c, "/v1/grants", map[string]string{"resource": req.Resource, "identity": string(req.Identity), "privilege": req.Privilege})
}

// CreateGrant       godoc
// @Summary   Create a grant
// @Tags      Grants
// @Security  AccessKey
// @Param     body  body  CreateGrantRequest  true  "Parameters"
// @Accept    json
// @Produce   json
// @Success   201  {array}  Grant
// @Router    /grants [post]
func (c Client) CreateGrant(req *CreateGrantRequest) (*Grant, error) {
	return post[CreateGrantRequest, Grant](c, "/v1/grants", req)
}

// DeleteGrant       godoc
// @Summary   Delete a grant
// @Tags      Grants
// @Security  AccessKey
// @Param     id  path  string  true  "grant ID"
// @Success   200
// @Router    /grants/{id} [delete]
func (c Client) DeleteGrant(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/grants/%s", id))
}

// @tag.name Destinations
// @tag.description Manage Destinations

// ListDestinations   godoc
// @Summary   List all destinations
// @Tags      Destinations
// @Security  AccessKey
// @Param     name       query  string  false  "destination name to filter by"
// @Param     unique_id  query  string  false  "unique destination id to filter by"
// @Produce   json
// @Success   200  {array}  Destination
// @Router    /destinations [get]
func (c Client) ListDestinations(req ListDestinationsRequest) ([]Destination, error) {
	return list[Destination](c, "/v1/destinations", map[string]string{"name": req.Name, "unique_id": req.UniqueID})
}

// CreateDestination  godoc
// @Summary   Create a destination
// @Tags      Destinations
// @Security  AccessKey
// @Param     body  body  CreateDestinationRequest  true  "Parameters"
// @Accept    json
// @Produce   json
// @Success   201  {array}  Destination
// @Router    /destinations [post]
func (c Client) CreateDestination(req *CreateDestinationRequest) (*Destination, error) {
	return post[CreateDestinationRequest, Destination](c, "/v1/destinations", req)
}

// UpdateDestination    godoc
// @Summary   Update a destination
// @Tags      Destinations
// @Security  AccessKey
// @Param     id    path  string                    true  "Unique ID"
// @Param     body  body  UpdateDestinationRequest  true  "Parameters"
// @Produce   json
// @Success   200  {object}  Destination
// @Router    /destinations/{id} [put]
func (c Client) UpdateDestination(req UpdateDestinationRequest) (*Destination, error) {
	return put[UpdateDestinationRequest, Destination](c, fmt.Sprintf("/v1/destinations/%s", req.ID.String()), &req)
}

// DeleteDestination    godoc
// @Summary   Delete a destination
// @Tags      Destinations
// @Security  AccessKey
// @Param     id  path  string  true  "Unique ID"
// @Produce   json
// @Success   200
// @Router    /destinations/{id} [delete]
func (c Client) DeleteDestination(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/destinations/%s", id))
}

func (c Client) ListAccessKeys(req ListAccessKeysRequest) ([]AccessKey, error) {
	return list[AccessKey](c, "/v1/access-keys", map[string]string{"machine_id": req.MachineID.String(), "name": req.Name})
}

func (c Client) CreateAccessKey(req *CreateAccessKeyRequest) (*CreateAccessKeyResponse, error) {
	return post[CreateAccessKeyRequest, CreateAccessKeyResponse](c, "/v1/access-keys", req)
}

func (c Client) DeleteAccessKey(id uid.ID) error {
	return delete(c, fmt.Sprintf("/v1/access-keys/%s", id))
}

func (c Client) ListMachines(req ListMachinesRequest) ([]Machine, error) {
	return list[Machine](c, "/v1/machines", map[string]string{"name": req.Name})
}

func (c Client) GetMachine(id uid.ID) (*Machine, error) {
	return get[Machine](c, fmt.Sprintf("/v1/machines/%s", id))
}

func (c Client) CreateMachine(req *CreateMachineRequest) (*Machine, error) {
	return post[CreateMachineRequest, Machine](c, "/v1/machines", req)
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

func (c Client) Introspect() (*Introspect, error) {
	return get[Introspect](c, "/v1/introspect")
}

// Login              godoc
// @Summary  Login to Infra
// @Tags     Login & Setup
// @Param    body  body  LoginRequest  true  "Parameters"
// @Accept   json
// @Produce  json
// @Success  200  {array}  LoginResponse
// @Router   /login [post]
func (c Client) Login(req *LoginRequest) (*LoginResponse, error) {
	return post[LoginRequest, LoginResponse](c, "/v1/login", req)
}

// Logout             godoc
// @Summary  Logout of Infra
// @Tags     Login & Setup
// @Success  200
// @Router   /logout [post]
func (c Client) Logout() error {
	_, err := post[EmptyRequest, EmptyResponse](c, "/v1/login", &EmptyRequest{})
	return err
}

// SetupRequired      godoc
// @Summary  Verify if Infra needs setup
// @Tags     Login & Setup
// @Accept   json
// @Produce  json
// @Success  200  {object}  SetupRequiredResponse
// @Router   /setup [get]
func (c Client) SetupRequired() (*SetupRequiredResponse, error) {
	return get[SetupRequiredResponse](c, "/v1/setup")
}

// Setup              godoc
// @Summary  Setup and initialize Infra
// @Tags     Login & Setup
// @Accept   json
// @Produce  json
// @Success  200  {array}  CreateAccessKeyResponse
// @Router   /setup [post]
func (c Client) Setup() (*CreateAccessKeyResponse, error) {
	return post[EmptyRequest, CreateAccessKeyResponse](c, "/v1/setup", &EmptyRequest{})
}

func (c Client) GetVersion() (*Version, error) {
	return get[Version](c, "/v1/version")
}

func partialText(body []byte, limit int) string {
	if len(body) <= limit {
		return string(body)
	}

	return string(body[:limit]) + "..."
}
