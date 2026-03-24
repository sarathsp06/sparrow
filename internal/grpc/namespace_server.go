package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/namespace"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/pkg/storage"
	pb "github.com/sarathsp06/sparrow/proto"
)

// NamespaceServer implements the NamespaceService gRPC service.
type NamespaceServer struct {
	pb.UnimplementedNamespaceServiceServer
	svc *namespace.Service
}

var _ pb.NamespaceServiceServer = (*NamespaceServer)(nil)

// NewNamespaceServer creates a new NamespaceServer.
func NewNamespaceServer(svc *namespace.Service) *NamespaceServer {
	return &NamespaceServer{svc: svc}
}

// ---- NamespaceService ----

func (s *NamespaceServer) CreateNamespace(ctx context.Context, req *pb.CreateNamespaceRequest) (*pb.CreateNamespaceResponse, error) {
	ns, err := s.svc.CreateNamespace(ctx, namespace.CreateNamespaceRequest{
		TenantID:    tenant.DefaultTenantID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "namespace %q already exists", req.GetName())
		}
		return nil, toGRPCError(ctx, err, "create namespace")
	}

	return &pb.CreateNamespaceResponse{
		Namespace: namespaceToProto(ns),
	}, nil
}

func (s *NamespaceServer) GetNamespace(ctx context.Context, req *pb.GetNamespaceRequest) (*pb.GetNamespaceResponse, error) {
	var ns *namespace.Namespace
	var err error

	if req.GetId() != "" {
		id, parseErr := uuid.Parse(req.GetId())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", parseErr)
		}
		ns, err = s.svc.GetNamespace(ctx, id)
	} else if req.GetName() != "" {
		ns, err = s.svc.GetNamespaceByName(ctx, tenant.DefaultTenantID, req.GetName())
	} else {
		return nil, status.Error(codes.InvalidArgument, "either id or name is required")
	}

	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "namespace not found")
		}
		return nil, toGRPCError(ctx, err, "get namespace")
	}

	return &pb.GetNamespaceResponse{
		Namespace: namespaceToProto(ns),
	}, nil
}

func (s *NamespaceServer) ListNamespaces(ctx context.Context, req *pb.ListNamespacesRequest) (*pb.ListNamespacesResponse, error) {
	limit, offset := paginationDefaults(req.GetPagination())

	namespaces, total, err := s.svc.ListNamespaces(ctx, tenant.DefaultTenantID, int(limit), int(offset))
	if err != nil {
		return nil, toGRPCError(ctx, err, "list namespaces")
	}

	pbNamespaces := make([]*pb.NamespaceResource, len(namespaces))
	for i, ns := range namespaces {
		pbNamespaces[i] = namespaceToProto(ns)
	}

	return &pb.ListNamespacesResponse{
		Namespaces: pbNamespaces,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(total),
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

func (s *NamespaceServer) UpdateNamespace(ctx context.Context, req *pb.UpdateNamespaceRequest2) (*pb.UpdateNamespaceResponse2, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", err)
	}

	ns, err := s.svc.UpdateNamespace(ctx, namespace.UpdateNamespaceRequest{
		ID:          id,
		TenantID:    tenant.DefaultTenantID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "namespace not found")
		}
		return nil, toGRPCError(ctx, err, "update namespace")
	}

	return &pb.UpdateNamespaceResponse2{
		Namespace: namespaceToProto(ns),
	}, nil
}

func (s *NamespaceServer) DeleteNamespace(ctx context.Context, req *pb.DeleteNamespaceRequest) (*pb.DeleteNamespaceResponse2, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", err)
	}

	if err := s.svc.DeleteNamespace(ctx, tenant.DefaultTenantID, id); err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "namespace not found")
		}
		return nil, toGRPCError(ctx, err, "delete namespace")
	}

	return &pb.DeleteNamespaceResponse2{}, nil
}

// ---- Conversion helpers ----

func namespaceToProto(ns *namespace.Namespace) *pb.NamespaceResource {
	if ns == nil {
		return nil
	}
	return &pb.NamespaceResource{
		Id:          ns.ID.String(),
		TenantId:    ns.TenantID.String(),
		Name:        ns.Name,
		Description: ns.Description,
		CreatedAt:   timestamppb.New(ns.CreatedAt),
		UpdatedAt:   timestamppb.New(ns.UpdatedAt),
	}
}
