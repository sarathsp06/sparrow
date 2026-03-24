package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

// NamespaceConnectServer implements the Connect-RPC handlers for NamespaceService
// by delegating to the gRPC server implementation.
type NamespaceConnectServer struct {
	namespaceService pb.NamespaceServiceServer
}

// NewNamespaceConnectServer creates a new Connect-RPC adapter for namespace services.
func NewNamespaceConnectServer(
	namespaceService pb.NamespaceServiceServer,
) *NamespaceConnectServer {
	return &NamespaceConnectServer{
		namespaceService: namespaceService,
	}
}

// ---- NamespaceService ----

func (s *NamespaceConnectServer) CreateNamespace(ctx context.Context, req *connect.Request[pb.CreateNamespaceRequest]) (*connect.Response[pb.CreateNamespaceResponse], error) {
	res, err := s.namespaceService.CreateNamespace(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) GetNamespace(ctx context.Context, req *connect.Request[pb.GetNamespaceRequest]) (*connect.Response[pb.GetNamespaceResponse], error) {
	res, err := s.namespaceService.GetNamespace(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) ListNamespaces(ctx context.Context, req *connect.Request[pb.ListNamespacesRequest]) (*connect.Response[pb.ListNamespacesResponse], error) {
	res, err := s.namespaceService.ListNamespaces(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) UpdateNamespace(ctx context.Context, req *connect.Request[pb.UpdateNamespaceRequest2]) (*connect.Response[pb.UpdateNamespaceResponse2], error) {
	res, err := s.namespaceService.UpdateNamespace(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) DeleteNamespace(ctx context.Context, req *connect.Request[pb.DeleteNamespaceRequest]) (*connect.Response[pb.DeleteNamespaceResponse2], error) {
	res, err := s.namespaceService.DeleteNamespace(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// Compile-time interface check
var _ pbconnect.NamespaceServiceHandler = (*NamespaceConnectServer)(nil)
