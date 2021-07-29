// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// V1Client is the client API for V1 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type V1Client interface {
	ListUsers(ctx context.Context, in *ListUsersRequest, opts ...grpc.CallOption) (*ListUsersResponse, error)
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*User, error)
	DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	ListDestinations(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListDestinationsResponse, error)
	CreateDestination(ctx context.Context, in *CreateDestinationRequest, opts ...grpc.CallOption) (*Destination, error)
	ListSources(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListSourcesResponse, error)
	CreateSource(ctx context.Context, in *CreateSourceRequest, opts ...grpc.CallOption) (*Source, error)
	DeleteSource(ctx context.Context, in *DeleteSourceRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	ListRoles(ctx context.Context, in *ListRolesRequest, opts ...grpc.CallOption) (*ListRolesResponse, error)
	CreateCred(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CreateCredResponse, error)
	ListApiKeys(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListApiKeyResponse, error)
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	Logout(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Signup(ctx context.Context, in *SignupRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	Status(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*StatusResponse, error)
	Version(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VersionResponse, error)
}

type v1Client struct {
	cc grpc.ClientConnInterface
}

func NewV1Client(cc grpc.ClientConnInterface) V1Client {
	return &v1Client{cc}
}

func (c *v1Client) ListUsers(ctx context.Context, in *ListUsersRequest, opts ...grpc.CallOption) (*ListUsersResponse, error) {
	out := new(ListUsersResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/ListUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*User, error) {
	out := new(User)
	err := c.cc.Invoke(ctx, "/v1.V1/CreateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/v1.V1/DeleteUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) ListDestinations(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListDestinationsResponse, error) {
	out := new(ListDestinationsResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/ListDestinations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) CreateDestination(ctx context.Context, in *CreateDestinationRequest, opts ...grpc.CallOption) (*Destination, error) {
	out := new(Destination)
	err := c.cc.Invoke(ctx, "/v1.V1/CreateDestination", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) ListSources(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListSourcesResponse, error) {
	out := new(ListSourcesResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/ListSources", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) CreateSource(ctx context.Context, in *CreateSourceRequest, opts ...grpc.CallOption) (*Source, error) {
	out := new(Source)
	err := c.cc.Invoke(ctx, "/v1.V1/CreateSource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) DeleteSource(ctx context.Context, in *DeleteSourceRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/v1.V1/DeleteSource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) ListRoles(ctx context.Context, in *ListRolesRequest, opts ...grpc.CallOption) (*ListRolesResponse, error) {
	out := new(ListRolesResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/ListRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) CreateCred(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CreateCredResponse, error) {
	out := new(CreateCredResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/CreateCred", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) ListApiKeys(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ListApiKeyResponse, error) {
	out := new(ListApiKeyResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/ListApiKeys", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) Logout(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/v1.V1/Logout", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) Signup(ctx context.Context, in *SignupRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/Signup", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) Status(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *v1Client) Version(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VersionResponse, error) {
	out := new(VersionResponse)
	err := c.cc.Invoke(ctx, "/v1.V1/Version", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// V1Server is the server API for V1 service.
// All implementations must embed UnimplementedV1Server
// for forward compatibility
type V1Server interface {
	ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error)
	CreateUser(context.Context, *CreateUserRequest) (*User, error)
	DeleteUser(context.Context, *DeleteUserRequest) (*emptypb.Empty, error)
	ListDestinations(context.Context, *emptypb.Empty) (*ListDestinationsResponse, error)
	CreateDestination(context.Context, *CreateDestinationRequest) (*Destination, error)
	ListSources(context.Context, *emptypb.Empty) (*ListSourcesResponse, error)
	CreateSource(context.Context, *CreateSourceRequest) (*Source, error)
	DeleteSource(context.Context, *DeleteSourceRequest) (*emptypb.Empty, error)
	ListRoles(context.Context, *ListRolesRequest) (*ListRolesResponse, error)
	CreateCred(context.Context, *emptypb.Empty) (*CreateCredResponse, error)
	ListApiKeys(context.Context, *emptypb.Empty) (*ListApiKeyResponse, error)
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
	Logout(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Signup(context.Context, *SignupRequest) (*LoginResponse, error)
	Status(context.Context, *emptypb.Empty) (*StatusResponse, error)
	Version(context.Context, *emptypb.Empty) (*VersionResponse, error)
	mustEmbedUnimplementedV1Server()
}

// UnimplementedV1Server must be embedded to have forward compatible implementations.
type UnimplementedV1Server struct {
}

func (UnimplementedV1Server) ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUsers not implemented")
}
func (UnimplementedV1Server) CreateUser(context.Context, *CreateUserRequest) (*User, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
func (UnimplementedV1Server) DeleteUser(context.Context, *DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}
func (UnimplementedV1Server) ListDestinations(context.Context, *emptypb.Empty) (*ListDestinationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListDestinations not implemented")
}
func (UnimplementedV1Server) CreateDestination(context.Context, *CreateDestinationRequest) (*Destination, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateDestination not implemented")
}
func (UnimplementedV1Server) ListSources(context.Context, *emptypb.Empty) (*ListSourcesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListSources not implemented")
}
func (UnimplementedV1Server) CreateSource(context.Context, *CreateSourceRequest) (*Source, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateSource not implemented")
}
func (UnimplementedV1Server) DeleteSource(context.Context, *DeleteSourceRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteSource not implemented")
}
func (UnimplementedV1Server) ListRoles(context.Context, *ListRolesRequest) (*ListRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRoles not implemented")
}
func (UnimplementedV1Server) CreateCred(context.Context, *emptypb.Empty) (*CreateCredResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCred not implemented")
}
func (UnimplementedV1Server) ListApiKeys(context.Context, *emptypb.Empty) (*ListApiKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListApiKeys not implemented")
}
func (UnimplementedV1Server) Login(context.Context, *LoginRequest) (*LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedV1Server) Logout(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
func (UnimplementedV1Server) Signup(context.Context, *SignupRequest) (*LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signup not implemented")
}
func (UnimplementedV1Server) Status(context.Context, *emptypb.Empty) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedV1Server) Version(context.Context, *emptypb.Empty) (*VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}
func (UnimplementedV1Server) mustEmbedUnimplementedV1Server() {}

// UnsafeV1Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to V1Server will
// result in compilation errors.
type UnsafeV1Server interface {
	mustEmbedUnimplementedV1Server()
}

func RegisterV1Server(s grpc.ServiceRegistrar, srv V1Server) {
	s.RegisterService(&V1_ServiceDesc, srv)
}

func _V1_ListUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).ListUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/ListUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).ListUsers(ctx, req.(*ListUsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_CreateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).CreateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/CreateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).CreateUser(ctx, req.(*CreateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_DeleteUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).DeleteUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/DeleteUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).DeleteUser(ctx, req.(*DeleteUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_ListDestinations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).ListDestinations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/ListDestinations",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).ListDestinations(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_CreateDestination_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateDestinationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).CreateDestination(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/CreateDestination",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).CreateDestination(ctx, req.(*CreateDestinationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_ListSources_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).ListSources(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/ListSources",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).ListSources(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_CreateSource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateSourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).CreateSource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/CreateSource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).CreateSource(ctx, req.(*CreateSourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_DeleteSource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteSourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).DeleteSource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/DeleteSource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).DeleteSource(ctx, req.(*DeleteSourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_ListRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).ListRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/ListRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).ListRoles(ctx, req.(*ListRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_CreateCred_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).CreateCred(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/CreateCred",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).CreateCred(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_ListApiKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).ListApiKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/ListApiKeys",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).ListApiKeys(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/Login",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).Login(ctx, req.(*LoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_Logout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).Logout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/Logout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).Logout(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_Signup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).Signup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/Signup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).Signup(ctx, req.(*SignupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).Status(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _V1_Version_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(V1Server).Version(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.V1/Version",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(V1Server).Version(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// V1_ServiceDesc is the grpc.ServiceDesc for V1 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var V1_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.V1",
	HandlerType: (*V1Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListUsers",
			Handler:    _V1_ListUsers_Handler,
		},
		{
			MethodName: "CreateUser",
			Handler:    _V1_CreateUser_Handler,
		},
		{
			MethodName: "DeleteUser",
			Handler:    _V1_DeleteUser_Handler,
		},
		{
			MethodName: "ListDestinations",
			Handler:    _V1_ListDestinations_Handler,
		},
		{
			MethodName: "CreateDestination",
			Handler:    _V1_CreateDestination_Handler,
		},
		{
			MethodName: "ListSources",
			Handler:    _V1_ListSources_Handler,
		},
		{
			MethodName: "CreateSource",
			Handler:    _V1_CreateSource_Handler,
		},
		{
			MethodName: "DeleteSource",
			Handler:    _V1_DeleteSource_Handler,
		},
		{
			MethodName: "ListRoles",
			Handler:    _V1_ListRoles_Handler,
		},
		{
			MethodName: "CreateCred",
			Handler:    _V1_CreateCred_Handler,
		},
		{
			MethodName: "ListApiKeys",
			Handler:    _V1_ListApiKeys_Handler,
		},
		{
			MethodName: "Login",
			Handler:    _V1_Login_Handler,
		},
		{
			MethodName: "Logout",
			Handler:    _V1_Logout_Handler,
		},
		{
			MethodName: "Signup",
			Handler:    _V1_Signup_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _V1_Status_Handler,
		},
		{
			MethodName: "Version",
			Handler:    _V1_Version_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "v1.proto",
}
