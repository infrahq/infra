package registry

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/registry/mocks"
	v1 "github.com/infrahq/infra/internal/v1"
	"github.com/infrahq/infra/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	kubernetesClient "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

type mockSecretReader struct{}

func NewMockSecretReader() kubernetes.SecretReader {
	return &mockSecretReader{}
}
func (msr *mockSecretReader) Get(secretName string, client *kubernetesClient.Clientset) (string, error) {
	return "foo", nil
}

func TestAuthInterceptorPublic(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/Status",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = authInterceptor(db)(context.Background(), "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorDefaultUnauthenticated(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	ctx := context.Background()

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorNoAuthorizationMetadata(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "random", "metadata")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorEmptyAuthorization(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorWrongAuthorizationFormat(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "hello")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorWrongBearerAuthorizationFormat(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer hello")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorInvalidToken(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+generate.RandString(TOKEN_LEN))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func addUser(db *gorm.DB, email string, password string, admin bool) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}
		var user User

		err := infraSource.CreateUser(tx, &user, email, password, admin)
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return errors.New("could not create token")
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	return token.Id, secret, nil
}

func TestAuthInterceptorValidToken(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorAdmin(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorAdminPass(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", true)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorInvalidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListRoles",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+generate.RandString(API_KEY_LEN)))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorValidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListRoles",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiKey.Key))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorValidApiKeyInvalidMethod(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiKey.Key))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestLoginMethodEmptyRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodNilInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodEmptyInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type:  v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraEmptyPassword(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Email: "test@test.com",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraEmptyEmail(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Password: "passw0rd",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraSuccess(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	server := &V1Server{db: db}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Email:    "test@test.com",
			Password: "passw0rd",
		},
	}

	res, err := server.Login(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")
}

func TestLoginMethodNilOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodEmptyOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOktaMissingDomainRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Code: "code",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOktaMissingCodeRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Domain: "testing.okta.com",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOkta(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var source Source
	source.Type = "okta"
	source.ApiToken = "test-api-token/apiToken"
	source.Domain = "test.okta.com"
	source.ClientId = "test-client-id"
	source.ClientSecret = "test-client-secret/clientSecret"
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}

	var user User
	source.CreateUser(db, &user, "test@test.com", "", false)
	if err != nil {
		t.Fatal(err)
	}

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test@test.com", nil)

	testSecretReader := NewMockSecretReader()
	testConfig := &rest.Config{
		Host: "https://localhost",
	}
	testK8s := &kubernetes.Kubernetes{Config: testConfig, SecretReader: testSecretReader}

	server := &V1Server{db: db, okta: testOkta, k8s: testK8s}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Domain: "test.okta.com",
			Code:   "testcode",
		},
	}

	res, err := server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")
}

func TestSignup(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	server := &V1Server{db: db}

	req := &v1.SignupRequest{
		Email:    "test@test.com",
		Password: "passw0rd",
	}

	res, err := server.Signup(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")

	var user User
	err = db.First(&user).Error
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, user.Admin, true)
	assert.Equal(t, user.Email, "test@test.com")
}

func TestSignupWithExistingAdmin(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
	}

	addUser(db, "existing@user.com", "passw0rd", true)

	server := &V1Server{db: db}

	req := &v1.SignupRequest{
		Email:    "admin@test.com",
		Password: "adminpassw0rd",
	}

	res, err := server.Signup(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
	assert.Nil(t, res)
}

func TestVersion(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
	}

	server := &V1Server{db: db}

	res, err := server.Version(context.Background(), &emptypb.Empty{})
	assert.Equal(t, status.Code(err), codes.OK)
	assert.Equal(t, res.Version, version.Version)
}

func TestVersionPublicAuth(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
	}

	server := &V1Server{db: db}

	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/Version",
	}

	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return server.Version(ctx, req.(*emptypb.Empty))
	}

	res, err := authInterceptor(db)(context.Background(), &emptypb.Empty{}, unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
	assert.Equal(t, res.(*v1.VersionResponse).Version, version.Version)
}

func TestListRolesForClusterReturnsRolesFromConfig(t *testing.T) {
	// this in memory DB is setup in the config test
	server := &V1Server{db: db}

	req := &v1.ListRolesRequest{
		DestinationId: clusterA.Id,
	}

	res, err := server.ListRoles(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.OK)

	returnedUserRoles := make(map[string][]*v1.User)
	for _, r := range res.Roles {
		returnedUserRoles[r.Name] = r.Users
	}

	// check default roles granted on user create
	assert.Equal(t, 3, len(returnedUserRoles["view"]))
	assert.True(t, containsUser(returnedUserRoles["view"], iosDevUser.Email))
	assert.True(t, containsUser(returnedUserRoles["view"], standardUser.Email))
	assert.True(t, containsUser(returnedUserRoles["view"], adminUser.Email))

	// roles from groups
	assert.Equal(t, 2, len(returnedUserRoles["writer"]))
	assert.True(t, containsUser(returnedUserRoles["writer"], iosDevUser.Email))
	assert.True(t, containsUser(returnedUserRoles["writer"], standardUser.Email))

	// roles from direct user assignment
	assert.Equal(t, 1, len(returnedUserRoles["admin"]))
	assert.True(t, containsUser(returnedUserRoles["admin"], adminUser.Email))
	assert.Equal(t, 1, len(returnedUserRoles["reader"]))
	assert.True(t, containsUser(returnedUserRoles["reader"], standardUser.Email))
}

func TestListRolesOnlyFindsForSpecificCluster(t *testing.T) {
	// this in memory DB is setup in the config test
	server := &V1Server{db: db}

	req := &v1.ListRolesRequest{
		DestinationId: clusterA.Id,
	}

	res, err := server.ListRoles(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.OK)

	unexpectedClusterIds := make(map[string]bool)
	for _, r := range res.Roles {
		if r.Destination.Id != clusterA.Id {
			unexpectedClusterIds[r.Destination.Id] = true
		}
	}
	if len(unexpectedClusterIds) != 0 {
		var unexpectedClusters []string
		for id := range unexpectedClusterIds {
			unexpectedClusters = append(unexpectedClusters, id)
		}
		t.Errorf("ListRoles response should only contain roles for the specified cluster ID. Only expected " + clusterA.Id + " but found " + strings.Join(unexpectedClusters, ", "))
	}
}

func TestListRolesForUnknownCluster(t *testing.T) {
	// this in memory DB is setup in the config test
	server := &V1Server{db: db}

	req := &v1.ListRolesRequest{
		DestinationId: "Unknown-Cluster-ID",
	}

	res, err := server.ListRoles(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.OK)

	assert.Equal(t, 0, len(res.Roles))
}

func containsUser(users []*v1.User, email string) bool {
	for _, u := range users {
		if u.Email == email {
			return true
		}
	}
	return false
}
