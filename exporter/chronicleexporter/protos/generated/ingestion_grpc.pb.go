// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.3
// source: ingestion.proto

package generated

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

// IngestionServiceV2Client is the client API for IngestionServiceV2 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IngestionServiceV2Client interface {
	// Submit a batch of events.
	BatchCreateEvents(ctx context.Context, in *BatchCreateEventsRequest, opts ...grpc.CallOption) (*BatchCreateEventsResponse, error)
	// Submit a batch of log entries.
	BatchCreateLogs(ctx context.Context, in *BatchCreateLogsRequest, opts ...grpc.CallOption) (*BatchCreateLogsResponse, error)
}

type ingestionServiceV2Client struct {
	cc grpc.ClientConnInterface
}

func NewIngestionServiceV2Client(cc grpc.ClientConnInterface) IngestionServiceV2Client {
	return &ingestionServiceV2Client{cc}
}

func (c *ingestionServiceV2Client) BatchCreateEvents(ctx context.Context, in *BatchCreateEventsRequest, opts ...grpc.CallOption) (*BatchCreateEventsResponse, error) {
	out := new(BatchCreateEventsResponse)
	err := c.cc.Invoke(ctx, "/malachite.ingestion.v2.IngestionServiceV2/BatchCreateEvents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ingestionServiceV2Client) BatchCreateLogs(ctx context.Context, in *BatchCreateLogsRequest, opts ...grpc.CallOption) (*BatchCreateLogsResponse, error) {
	out := new(BatchCreateLogsResponse)
	err := c.cc.Invoke(ctx, "/malachite.ingestion.v2.IngestionServiceV2/BatchCreateLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IngestionServiceV2Server is the server API for IngestionServiceV2 service.
// All implementations must embed UnimplementedIngestionServiceV2Server
// for forward compatibility
type IngestionServiceV2Server interface {
	// Submit a batch of events.
	BatchCreateEvents(context.Context, *BatchCreateEventsRequest) (*BatchCreateEventsResponse, error)
	// Submit a batch of log entries.
	BatchCreateLogs(context.Context, *BatchCreateLogsRequest) (*BatchCreateLogsResponse, error)
	mustEmbedUnimplementedIngestionServiceV2Server()
}

// UnimplementedIngestionServiceV2Server must be embedded to have forward compatible implementations.
type UnimplementedIngestionServiceV2Server struct {
}

func (UnimplementedIngestionServiceV2Server) BatchCreateEvents(context.Context, *BatchCreateEventsRequest) (*BatchCreateEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchCreateEvents not implemented")
}
func (UnimplementedIngestionServiceV2Server) BatchCreateLogs(context.Context, *BatchCreateLogsRequest) (*BatchCreateLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchCreateLogs not implemented")
}
func (UnimplementedIngestionServiceV2Server) mustEmbedUnimplementedIngestionServiceV2Server() {}

// UnsafeIngestionServiceV2Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IngestionServiceV2Server will
// result in compilation errors.
type UnsafeIngestionServiceV2Server interface {
	mustEmbedUnimplementedIngestionServiceV2Server()
}

func RegisterIngestionServiceV2Server(s grpc.ServiceRegistrar, srv IngestionServiceV2Server) {
	s.RegisterService(&IngestionServiceV2_ServiceDesc, srv)
}

func _IngestionServiceV2_BatchCreateEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchCreateEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IngestionServiceV2Server).BatchCreateEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/malachite.ingestion.v2.IngestionServiceV2/BatchCreateEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IngestionServiceV2Server).BatchCreateEvents(ctx, req.(*BatchCreateEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IngestionServiceV2_BatchCreateLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchCreateLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IngestionServiceV2Server).BatchCreateLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/malachite.ingestion.v2.IngestionServiceV2/BatchCreateLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IngestionServiceV2Server).BatchCreateLogs(ctx, req.(*BatchCreateLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// IngestionServiceV2_ServiceDesc is the grpc.ServiceDesc for IngestionServiceV2 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IngestionServiceV2_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "malachite.ingestion.v2.IngestionServiceV2",
	HandlerType: (*IngestionServiceV2Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BatchCreateEvents",
			Handler:    _IngestionServiceV2_BatchCreateEvents_Handler,
		},
		{
			MethodName: "BatchCreateLogs",
			Handler:    _IngestionServiceV2_BatchCreateLogs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ingestion.proto",
}
