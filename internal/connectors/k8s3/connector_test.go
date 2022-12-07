package k8s3

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/uid"
)

func TestCredentialRequestReconciler(t *testing.T) {
	ctx := context.Background()
	fakeApi := FakeAPI{t: t}
	fakeK8s := FakeK8s{t: t}
	con := &k8sConnector{
		k8s:    fakeK8s,
		client: fakeApi,
		destination: api.Destination{
			Name: "test",
		},
		options: Options{},
	}

	resp := pollForCredentialRequestsOnce(ctx, con, 0)
	assert.Assert(t, resp != nil)
}

type FakeAPI struct {
	t *testing.T
}

func (f FakeAPI) ListGrants(ctx context.Context, req api.ListGrantsRequest) (*api.ListResponse[api.Grant], error) {
	return nil, nil
}
func (f FakeAPI) ListDestinations(ctx context.Context, req api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	return nil, nil
}
func (f FakeAPI) CreateDestination(ctx context.Context, req *api.CreateDestinationRequest) (*api.Destination, error) {
	return nil, nil
}
func (f FakeAPI) UpdateDestination(ctx context.Context, req api.UpdateDestinationRequest) (*api.Destination, error) {
	return nil, nil
}
func (f FakeAPI) ListCredentialRequests(ctx context.Context, req api.ListCredentialRequest) (*api.ListCredentialRequestResponse, error) {
	return &api.ListCredentialRequestResponse{
		Items: []api.CredentialRequest{
			{
				ID:             10,
				OrganizationID: 2,
				UserID:         3,
				Destination:    "test",
			},
		},
		MaxUpdateIndex: 1,
	}, nil
}
func (f FakeAPI) UpdateCredentialRequest(ctx context.Context, r *api.UpdateCredentialRequest) (*api.EmptyResponse, error) {
	assert.Equal(f.t, int64(r.ID), int64(10))
	assert.Equal(f.t, int(r.OrganizationID), 2)
	assert.Equal(f.t, r.BearerToken, "abc123")
	return nil, nil
}
func (f FakeAPI) GetGroup(ctx context.Context, id uid.ID) (*api.Group, error) { return nil, nil }
func (f FakeAPI) GetUser(ctx context.Context, id uid.ID) (*api.User, error) {
	return &api.User{ID: 3, Name: "testuser@example.com"}, nil
}

type FakeK8s struct{ t *testing.T }

func (f FakeK8s) Namespaces() ([]string, error)                                        { return nil, nil }
func (f FakeK8s) ClusterRoles() ([]string, error)                                      { return nil, nil }
func (f FakeK8s) IsServiceTypeClusterIP() (bool, error)                                { return false, nil }
func (f FakeK8s) DirectEndpoint() (string, int, []byte, error)                         { return "", 0, nil, nil }
func (f FakeK8s) UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error { return nil }
func (f FakeK8s) UpdateRoleBindings(subjects map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) error {
	return nil
}
func (f FakeK8s) CreateServiceAccount(username string) (*corev1.ServiceAccount, error) {
	assert.Equal(f.t, username, "testuser@example.com")
	return &corev1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: username}}, nil
}
func (f FakeK8s) CreateServiceAccountToken(username string) (*authenticationv1.TokenRequest, error) {
	return &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: "abc123"}}, nil
}
