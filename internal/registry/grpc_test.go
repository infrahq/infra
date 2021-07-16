package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/infrahq/infra/internal/generate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestAuthInterceptorPublic(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/Status",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+generate.RandString(TOKEN_LEN))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func addUser(db *gorm.DB, admin bool) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}
		var user User

		err := infraSource.CreateUser(tx, &user, "test@email.com", "password", admin)
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
		t.Error(err)
	}

	id, secret, err := addUser(db, false)
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
		t.Error(err)
	}

	id, secret, err := addUser(db, false)
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
		t.Error(err)
	}

	id, secret, err := addUser(db, true)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorInvalidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListPermissions",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+generate.RandString(API_KEY_LEN)))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorValidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListPermissions",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
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
		t.Error(err)
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
