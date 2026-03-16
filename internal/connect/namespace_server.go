package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

// NamespaceConnectServer implements the Connect-RPC handlers for NamespaceService
// and NamespaceMembershipService by delegating to the gRPC server implementations.
type NamespaceConnectServer struct {
	namespaceService           pb.NamespaceServiceServer
	namespaceMembershipService pb.NamespaceMembershipServiceServer
}

// NewNamespaceConnectServer creates a new Connect-RPC adapter for namespace services.
func NewNamespaceConnectServer(
	namespaceService pb.NamespaceServiceServer,
	namespaceMembershipService pb.NamespaceMembershipServiceServer,
) *NamespaceConnectServer {
	return &NamespaceConnectServer{
		namespaceService:           namespaceService,
		namespaceMembershipService: namespaceMembershipService,
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

// ---- NamespaceMembershipService ----

func (s *NamespaceConnectServer) AssignNamespaceRole(ctx context.Context, req *connect.Request[pb.AssignNamespaceRoleRequest]) (*connect.Response[pb.AssignNamespaceRoleResponse], error) {
	res, err := s.namespaceMembershipService.AssignNamespaceRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) RemoveNamespaceRole(ctx context.Context, req *connect.Request[pb.RemoveNamespaceRoleRequest]) (*connect.Response[pb.RemoveNamespaceRoleResponse], error) {
	res, err := s.namespaceMembershipService.RemoveNamespaceRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) ListNamespaceMembers(ctx context.Context, req *connect.Request[pb.ListNamespaceMembersRequest]) (*connect.Response[pb.ListNamespaceMembersResponse], error) {
	res, err := s.namespaceMembershipService.ListNamespaceMembers(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *NamespaceConnectServer) GetUserNamespaces(ctx context.Context, req *connect.Request[pb.GetUserNamespacesRequest]) (*connect.Response[pb.GetUserNamespacesResponse], error) {
	res, err := s.namespaceMembershipService.GetUserNamespaces(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// Compile-time interface checks
var _ pbconnect.NamespaceServiceHandler = (*NamespaceConnectServer)(nil)
var _ pbconnect.NamespaceMembershipServiceHandler = (*NamespaceConnectServer)(nil)
