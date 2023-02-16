// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: db.proto

package service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DatabaseTestClient is the client API for DatabaseTest service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DatabaseTestClient interface {
	GetUserById(ctx context.Context, in *UserByIdRequest, opts ...grpc.CallOption) (*User, error)
	AddOrUpdateUser(ctx context.Context, in *User, opts ...grpc.CallOption) (*UpdateResponse, error)
}

type databaseTestClient struct {
	cc grpc.ClientConnInterface
}

func NewDatabaseTestClient(cc grpc.ClientConnInterface) DatabaseTestClient {
	return &databaseTestClient{cc}
}

func (c *databaseTestClient) GetUserById(ctx context.Context, in *UserByIdRequest, opts ...grpc.CallOption) (*User, error) {
	out := new(User)
	err := c.cc.Invoke(ctx, "/service.DatabaseTest/GetUserById", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseTestClient) AddOrUpdateUser(ctx context.Context, in *User, opts ...grpc.CallOption) (*UpdateResponse, error) {
	out := new(UpdateResponse)
	err := c.cc.Invoke(ctx, "/service.DatabaseTest/AddOrUpdateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DatabaseTestServer is the server API for DatabaseTest service.
// All implementations must embed UnimplementedDatabaseTestServer
// for forward compatibility
type DatabaseTestServer interface {
	GetUserById(context.Context, *UserByIdRequest) (*User, error)
	AddOrUpdateUser(context.Context, *User) (*UpdateResponse, error)
	mustEmbedUnimplementedDatabaseTestServer()
}

// UnimplementedDatabaseTestServer must be embedded to have forward compatible implementations.
type UnimplementedDatabaseTestServer struct {
}

func (UnimplementedDatabaseTestServer) GetUserById(context.Context, *UserByIdRequest) (*User, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserById not implemented")
}
func (UnimplementedDatabaseTestServer) AddOrUpdateUser(context.Context, *User) (*UpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddOrUpdateUser not implemented")
}
func (UnimplementedDatabaseTestServer) mustEmbedUnimplementedDatabaseTestServer() {}

// UnsafeDatabaseTestServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DatabaseTestServer will
// result in compilation errors.
type UnsafeDatabaseTestServer interface {
	mustEmbedUnimplementedDatabaseTestServer()
}

func RegisterDatabaseTestServer(s grpc.ServiceRegistrar, srv DatabaseTestServer) {
	s.RegisterService(&DatabaseTest_ServiceDesc, srv)
}

func _DatabaseTest_GetUserById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserByIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseTestServer).GetUserById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.DatabaseTest/GetUserById",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseTestServer).GetUserById(ctx, req.(*UserByIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseTest_AddOrUpdateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseTestServer).AddOrUpdateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.DatabaseTest/AddOrUpdateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseTestServer).AddOrUpdateUser(ctx, req.(*User))
	}
	return interceptor(ctx, in, info, handler)
}

// DatabaseTest_ServiceDesc is the grpc.ServiceDesc for DatabaseTest service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DatabaseTest_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "service.DatabaseTest",
	HandlerType: (*DatabaseTestServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUserById",
			Handler:    _DatabaseTest_GetUserById_Handler,
		},
		{
			MethodName: "AddOrUpdateUser",
			Handler:    _DatabaseTest_AddOrUpdateUser_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "db.proto",
}
