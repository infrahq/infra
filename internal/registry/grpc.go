package registry

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/infrahq/infra/internal/generate"
	v1 "github.com/infrahq/infra/internal/v1"
	"github.com/infrahq/infra/internal/version"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
)

type V1Server struct {
	v1.UnimplementedV1Server
	db   *gorm.DB
	okta Okta
}

var publicMethods = map[string]bool{
	"/v1.V1/Status":      true,
	"/v1.V1/ListSources": true,
	"/v1.V1/Login":       true,
	"/v1.V1/Signup":      true,
}

var tokenAuthMethods = map[string]bool{
	"/v1.V1/ListUsers":        true,
	"/v1.V1/CreateUser":       true,
	"/v1.V1/DeleteUser":       true,
	"/v1.V1/ListDestinations": true,
	"/v1.V1/CreateSource":     true,
	"/v1.V1/DeleteSource":     true,
	"/v1.V1/ListPermissions":  true,
	"/v1.V1/CreateCred":       true,
	"/v1.V1/ListApiKeys":      true,
	"/v1.V1/Logout":           true,
}

var tokenAuthAdminMethods = map[string]bool{
	"/v1.V1/CreateUser":   true,
	"/v1.V1/DeleteUser":   true,
	"/v1.V1/CreateSource": true,
	"/v1.V1/DeleteSource": true,
	"/v1.V1/ListApiKeys":  true,
}

var apiKeyAuthMethods = map[string]bool{
	"/v1.V1/CreateDestination": true,
	"/v1.V1/ListPermissions":   true,
}

type UserIdContextKey struct{}

func authInterceptor(db *gorm.DB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
		}

		authorization, ok := md["authorization"]
		if !ok || len(authorization) == 0 {
			grpc_zap.Extract(ctx).Debug("No authorization specified in auth interceptor")
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
		}

		raw := strings.Replace(authorization[0], "Bearer ", "", -1)

		if raw == "" {
			grpc_zap.Extract(ctx).Debug("No bearer token recieved")
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
		}

		// TODO (https://github.com/infrahq/infra/issues/60): use a token prefix or separate routes instead
		// of using the length to determine the token kind
		switch len(raw) {
		case TOKEN_LEN:
			if !tokenAuthMethods[info.FullMethod] {
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}

			id := raw[0:ID_LEN]
			secret := raw[ID_LEN:TOKEN_LEN]

			var token Token
			if err := db.First(&token, &Token{Id: id}).Error; err != nil {
				grpc_zap.Extract(ctx).Debug("Invalid token presented to the auth interceptor")
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}

			if err := token.CheckSecret(secret); err != nil {
				grpc_zap.Extract(ctx).Debug("Invalid secret on token presented to the auth interceptor")
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}

			if tokenAuthAdminMethods[info.FullMethod] {
				var user User
				if err := db.First(&user, &User{Id: token.UserId}).Error; err != nil {
					grpc_zap.Extract(ctx).Debug("Could not find user")
					return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
				}

				if !user.Admin {
					grpc_zap.Extract(ctx).Debug("Unauthorized user attempted to authenticate without admin privilege")
					return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
				}
			}

			return handler(context.WithValue(ctx, UserIdContextKey{}, token.UserId), req)
		case API_KEY_LEN:
			if !apiKeyAuthMethods[info.FullMethod] {
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}

			var apiKey ApiKey
			if db.First(&apiKey, &ApiKey{Key: raw}).Error != nil {
				grpc_zap.Extract(ctx).Debug("Invalid API key token presented")
				return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
			}

			return handler(ctx, req)
		default:
			grpc_zap.Extract(ctx).Debug("Unknown token type presented")
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
		}
	}
}

func (v *V1Server) createJWT(email string) (string, time.Time, error) {
	var settings Settings
	err := v.db.First(&settings).Error
	if err != nil {
		return "", time.Time{}, err
	}

	var key jose.JSONWebKey
	err = key.UnmarshalJSON(settings.PrivateJWK)
	if err != nil {
		return "", time.Time{}, err
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(time.Minute * 5)

	cl := jwt.Claims{
		Issuer:   "infra",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	custom := struct {
		Email string `json:"email"`
		Nonce string `json:"nonce"`
	}{
		email,
		generate.RandString(10),
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", time.Time{}, err
	}

	return raw, expiry, nil
}

func dbToProtoUser(in *User) *v1.User {
	return &v1.User{
		Id:      in.Id,
		Created: in.Created,
		Updated: in.Updated,
		Email:   in.Email,
		Admin:   in.Admin,
	}
}

func (v *V1Server) ListUsers(ctx context.Context, in *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	q := v.db

	if in.Email != "" {
		q = q.Where("email = ?", in.Email)
	}

	var users []User
	err := q.Find(&users).Error
	if err != nil {
		return nil, err
	}

	res := &v1.ListUsersResponse{}
	for _, u := range users {
		res.Users = append(res.Users, dbToProtoUser(&u))
	}

	return res, nil
}

func (v *V1Server) CreateUser(ctx context.Context, in *v1.CreateUserRequest) (*v1.User, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	var user User
	err := v.db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}

		if tx.Model(&infraSource).Where(&User{Email: in.Email}).Association("Users").Count() > 0 {
			return errors.New("user with this email already exists")
		}

		err := infraSource.CreateUser(tx, &user, in.Email, in.Password, false)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return dbToProtoUser(&user), nil
}

func (v *V1Server) DeleteUser(ctx context.Context, in *v1.DeleteUserRequest) (*empty.Empty, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	userId, ok := ctx.Value(UserIdContextKey{}).(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// TODO: check for self
	err := v.db.Transaction(func(tx *gorm.DB) error {
		if userId == in.Id {
			return status.Errorf(codes.InvalidArgument, "cannot delete self")
		}

		var user User
		err := tx.First(&user, "id = ?", in.Id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user does not exist")
		}

		if err != nil {
			return err
		}

		if tx.Model(&user).Where(&Source{Type: SOURCE_TYPE_INFRA}).Association("Sources").Count() == 0 {
			return errors.New("user managed by external identity source")
		}

		var count int64
		err = tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			return err
		}

		if user.Admin && count == 1 {
			return errors.New("cannot delete last admin user")
		}

		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}

		err = infraSource.DeleteUser(tx, &user)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func dbToProtoSource(in *Source) *v1.Source {
	out := &v1.Source{
		Id:      in.Id,
		Created: in.Created,
		Updated: in.Updated,
	}

	switch in.Type {
	case SOURCE_TYPE_OKTA:
		out.Type = v1.SourceType_OKTA
		out.Okta = &v1.Source_Okta{
			Domain:   in.OktaDomain,
			ClientId: in.OktaClientId,
		}
	}

	return out
}

func (v *V1Server) ListSources(context.Context, *emptypb.Empty) (*v1.ListSourcesResponse, error) {
	var sources []Source
	err := v.db.Transaction(func(tx *gorm.DB) error {
		return tx.Find(&sources).Error
	})
	if err != nil {
		return nil, err
	}

	res := &v1.ListSourcesResponse{}
	for _, s := range sources {
		res.Sources = append(res.Sources, dbToProtoSource(&s))
	}

	return res, nil
}

func (v *V1Server) CreateSource(ctx context.Context, in *v1.CreateSourceRequest) (*v1.Source, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	var source Source
	switch in.Type {
	case *v1.SourceType_OKTA.Enum():
		if err := in.Okta.ValidateAll(); err != nil {
			return nil, err
		}
		if err := v.okta.ValidateOktaConnection(in.Okta.Domain, in.Okta.ClientId, in.Okta.ApiToken); err != nil {
			return nil, err
		}

		source.Type = "okta"
		source.OktaApiToken = in.Okta.ApiToken
		source.OktaDomain = in.Okta.Domain
		source.OktaClientId = in.Okta.ClientId
		source.OktaClientSecret = in.Okta.ClientSecret

		if err := v.db.Create(&source).Error; err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid source type")
	}

	if err := source.SyncUsers(v.db, v.okta); err != nil {
		return nil, err
	}

	return dbToProtoSource(&source), nil
}

func (v *V1Server) DeleteSource(ctx context.Context, in *v1.DeleteSourceRequest) (*emptypb.Empty, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	err := v.db.Transaction(func(tx *gorm.DB) error {
		var source Source
		count := tx.First(&source, &Source{Id: in.Id}).RowsAffected
		if count == 0 {
			return errors.New("no such source")
		}

		if source.Type == SOURCE_TYPE_INFRA {
			return errors.New("cannot delete infra source")
		}

		return tx.Delete(&source).Error
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func dbToProtoDestination(in *Destination) *v1.Destination {
	out := &v1.Destination{
		Id:      in.Id,
		Created: in.Created,
		Updated: in.Updated,
		Name:    in.Name,
	}

	switch in.Type {
	case DESTINATION_TYPE_KUBERNERNETES:
		out.Type = v1.DestinationType_KUBERNETES
		out.Kubernetes = &v1.Destination_Kubernetes{
			Ca:        in.KubernetesCa,
			Endpoint:  in.KubernetesEndpoint,
			Namespace: in.KubernetesNamespace,
			SaToken:   in.KubernetesSaToken,
		}
	}
	return out
}

func (v *V1Server) ListDestinations(ctx context.Context, _ *emptypb.Empty) (*v1.ListDestinationsResponse, error) {
	var destinations []Destination
	err := v.db.Find(&destinations).Error
	if err != nil {
		return nil, err
	}

	res := &v1.ListDestinationsResponse{}
	for _, d := range destinations {
		res.Destinations = append(res.Destinations, dbToProtoDestination(&d))
	}

	return res, nil
}

func (v *V1Server) CreateDestination(ctx context.Context, in *v1.CreateDestinationRequest) (*v1.Destination, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	var model Destination
	err := v.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&Destination{Name: in.Name}).FirstOrCreate(&model)
		if result.Error != nil {
			return result.Error
		}

		model.Name = in.Name

		switch in.Type {
		case v1.DestinationType_KUBERNETES:
			if err := in.Kubernetes.ValidateAll(); err != nil {
				return err
			}
			model.Type = DESTINATION_TYPE_KUBERNERNETES
			model.KubernetesCa = in.Kubernetes.Ca
			model.KubernetesEndpoint = in.Kubernetes.Endpoint
			model.KubernetesNamespace = in.Kubernetes.Namespace
			model.KubernetesSaToken = in.Kubernetes.SaToken
		}

		return tx.Save(&model).Error
	})
	if err != nil {
		return nil, err
	}

	return dbToProtoDestination(&model), nil
}

func dbToProtoPermission(in *Permission) *v1.Permission {
	return &v1.Permission{
		Id:          in.Id,
		Created:     in.Created,
		Updated:     in.Updated,
		Role:        in.Role,
		User:        dbToProtoUser(&in.User),
		Destination: dbToProtoDestination(&in.Destination),
	}
}

func (v *V1Server) ListPermissions(ctx context.Context, in *v1.ListPermissionsRequest) (*v1.ListPermissionsResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	var permissions []Permission
	err := v.db.Preload("User").Preload("Destination").Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	res := &v1.ListPermissionsResponse{}
	for _, p := range permissions {
		res.Permissions = append(res.Permissions, dbToProtoPermission(&p))
	}

	return res, nil
}

func (v *V1Server) CreateCred(ctx context.Context, in *emptypb.Empty) (*v1.CreateCredResponse, error) {
	userId, ok := ctx.Value(UserIdContextKey{}).(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	var user User
	err := v.db.Where(&User{Id: userId}).Find(&user).Error
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user")
	}

	token, expiry, err := v.createJWT(user.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not generate cred")
	}

	return &v1.CreateCredResponse{
		Token:   token,
		Expires: expiry.Unix(),
	}, nil
}

func (v *V1Server) ListApiKeys(ctx context.Context, in *emptypb.Empty) (*v1.ListApiKeyResponse, error) {
	var apiKeys []ApiKey
	err := v.db.Find(&apiKeys).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	res := &v1.ListApiKeyResponse{}
	for _, ak := range apiKeys {
		res.ApiKeys = append(res.ApiKeys, &v1.ApiKey{
			Id:      ak.Id,
			Created: ak.Created,
			Updated: ak.Updated,
			Name:    ak.Name,
			Key:     ak.Key,
		})
	}

	return res, nil
}

func (v *V1Server) Login(ctx context.Context, in *v1.LoginRequest) (*v1.LoginResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var user User
	var token Token

	switch in.Type {
	case v1.SourceType_OKTA:
		if in.Okta == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing okta login information")
		}

		var source Source
		if err := v.db.Where(&Source{Type: SOURCE_TYPE_OKTA, OktaDomain: in.Okta.Domain}).First(&source).Error; err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid okta login information")
		}

		email, err := v.okta.EmailFromCode(
			in.Okta.Code,
			source.OktaDomain,
			source.OktaClientId,
			source.OktaClientSecret,
		)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid okta login information")
		}

		err = v.db.Where("email = ?", email).First(&user).Error
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid okta login information")
		}
	case v1.SourceType_INFRA:
		if in.Infra == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing login information")
		}

		if err := v.db.Where("email = ?", in.Infra.Email).First(&user).Error; err != nil {
			grpc_zap.Extract(ctx).Debug("User failed to login with unknown email")
			return nil, status.Errorf(codes.Unauthenticated, "invalid login information")
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(in.Infra.Password)); err != nil {
			grpc_zap.Extract(ctx).Debug("User failed to login with invalid password")
			return nil, status.Errorf(codes.Unauthenticated, "invalid login information")
		}
	default:
		return nil, status.Errorf(codes.Unauthenticated, "invalid login type")
	}

	secret, err := NewToken(v.db, user.Id, &token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create token")
	}

	return &v1.LoginResponse{Token: token.Id + secret}, nil
}

func (v *V1Server) Logout(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	userId, ok := ctx.Value(UserIdContextKey{}).(string)
	if !ok {
		grpc_zap.Extract(ctx).Debug("Could not logout user, user ID not found")
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if err := v.db.Where(&Token{UserId: userId}).Delete(&Token{}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (v *V1Server) Signup(ctx context.Context, in *v1.SignupRequest) (*v1.LoginResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, err
	}

	var token Token
	var secret string
	err := v.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
		if err != nil {
			grpc_zap.Extract(ctx).Debug("Could not lookup admin users in the database")
			return status.Errorf(codes.Unauthenticated, "unauthorized")
		}

		if count > 0 {
			return status.Errorf(codes.InvalidArgument, "admin user already exists")
		}

		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		var user User
		if err := infraSource.CreateUser(tx, &user, in.Email, in.Password, true); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.LoginResponse{Token: token.Id + secret}, nil
}

func (v *V1Server) Status(ctx context.Context, in *emptypb.Empty) (*v1.StatusResponse, error) {
	var count int64
	err := v.db.Where(&User{Admin: true}).Find(&[]User{}).Count(&count).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve status")
	}

	return &v1.StatusResponse{Admin: count > 0}, nil
}

func (v *V1Server) Version(ctx context.Context, in *emptypb.Empty) (*v1.VersionResponse, error) {
	return &v1.VersionResponse{Version: version.Version}, nil
}
