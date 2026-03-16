package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/internal/namespace"
	"github.com/sarathsp06/sparrow/pkg/storage"
	pb "github.com/sarathsp06/sparrow/proto"
)

// NamespaceServer implements the NamespaceService and NamespaceMembershipService gRPC services.
type NamespaceServer struct {
	pb.UnimplementedNamespaceServiceServer
	pb.UnimplementedNamespaceMembershipServiceServer
	svc *namespace.Service
}

var _ pb.NamespaceServiceServer = (*NamespaceServer)(nil)
var _ pb.NamespaceMembershipServiceServer = (*NamespaceServer)(nil)

// NewNamespaceServer creates a new NamespaceServer.
func NewNamespaceServer(svc *namespace.Service) *NamespaceServer {
	return &NamespaceServer{svc: svc}
}

// ---- NamespaceService ----

func (s *NamespaceServer) CreateNamespace(ctx context.Context, req *pb.CreateNamespaceRequest) (*pb.CreateNamespaceResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermNamespaceCreate, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	ns, err := s.svc.CreateNamespace(ctx, namespace.CreateNamespaceRequest{
		TenantID:    info.TenantID,
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
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermNamespaceRead, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	var ns *namespace.Namespace
	var err error

	if req.GetId() != "" {
		id, parseErr := uuid.Parse(req.GetId())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", parseErr)
		}
		ns, err = s.svc.GetNamespace(ctx, id)
	} else if req.GetName() != "" {
		ns, err = s.svc.GetNamespaceByName(ctx, info.TenantID, req.GetName())
	} else {
		return nil, status.Error(codes.InvalidArgument, "either id or name is required")
	}

	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "namespace not found")
		}
		return nil, toGRPCError(ctx, err, "get namespace")
	}

	// Verify tenant ownership
	if ns.TenantID != info.TenantID && !info.IsPlatformAdmin {
		return nil, status.Error(codes.NotFound, "namespace not found")
	}

	// If user has namespace memberships, verify they can access this namespace
	if len(info.NamespaceRoles) > 0 && !info.IsPlatformAdmin {
		if _, ok := info.NamespaceRoles[ns.Name]; !ok {
			return nil, status.Error(codes.PermissionDenied, "no access to this namespace")
		}
	}

	return &pb.GetNamespaceResponse{
		Namespace: namespaceToProto(ns),
	}, nil
}

func (s *NamespaceServer) ListNamespaces(ctx context.Context, req *pb.ListNamespacesRequest) (*pb.ListNamespacesResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermNamespaceRead, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	limit, offset := paginationDefaults(req.GetPagination())

	namespaces, total, err := s.svc.ListNamespaces(ctx, info.TenantID, int(limit), int(offset))
	if err != nil {
		return nil, toGRPCError(ctx, err, "list namespaces")
	}

	// If user has namespace memberships, filter to only their accessible namespaces
	accessible := info.AccessibleNamespaces()
	if accessible != nil {
		accessibleSet := make(map[string]bool, len(accessible))
		for _, ns := range accessible {
			accessibleSet[ns] = true
		}
		filtered := make([]*namespace.Namespace, 0, len(namespaces))
		for _, ns := range namespaces {
			if accessibleSet[ns.Name] {
				filtered = append(filtered, ns)
			}
		}
		namespaces = filtered
		total = len(filtered)
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
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermNamespaceUpdate, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", err)
	}

	// If user has namespace memberships, verify they have admin on this namespace
	if len(info.NamespaceRoles) > 0 && !info.IsPlatformAdmin {
		// Need to look up the namespace to check the name
		existing, lookupErr := s.svc.GetNamespace(ctx, id)
		if lookupErr != nil {
			if storage.IsNotFound(lookupErr) {
				return nil, status.Error(codes.NotFound, "namespace not found")
			}
			return nil, toGRPCError(ctx, lookupErr, "get namespace")
		}
		if err := info.Require(auth.PermNamespaceUpdate, existing.Name); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	ns, err := s.svc.UpdateNamespace(ctx, namespace.UpdateNamespaceRequest{
		ID:          id,
		TenantID:    info.TenantID,
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
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermNamespaceDelete, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid namespace ID: %v", err)
	}

	if err := s.svc.DeleteNamespace(ctx, info.TenantID, id); err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "namespace not found")
		}
		return nil, toGRPCError(ctx, err, "delete namespace")
	}

	return &pb.DeleteNamespaceResponse2{}, nil
}

// ---- NamespaceMembershipService ----

func (s *NamespaceServer) AssignNamespaceRole(ctx context.Context, req *pb.AssignNamespaceRoleRequest) (*pb.AssignNamespaceRoleResponse, error) {
	info := auth.MustFromContext(ctx)

	// Check membership management permission — namespace-scoped if the caller
	// has memberships, otherwise tenant-scoped.
	if len(info.NamespaceRoles) > 0 && !info.IsPlatformAdmin {
		if err := info.Require(auth.PermNamespaceMembershipManage, req.GetNamespace()); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	} else {
		if err := info.Require(auth.PermNamespaceMembershipManage, ""); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	m, err := s.svc.AssignNamespaceRole(ctx, namespace.AssignMembershipRequest{
		TenantID:  info.TenantID,
		SubjectID: req.GetSubjectId(),
		Namespace: req.GetNamespace(),
		Role:      auth.Role(req.GetRole()),
	})
	if err != nil {
		return nil, toGRPCError(ctx, err, "assign namespace role")
	}

	return &pb.AssignNamespaceRoleResponse{
		Membership: membershipToProto(m),
	}, nil
}

func (s *NamespaceServer) RemoveNamespaceRole(ctx context.Context, req *pb.RemoveNamespaceRoleRequest) (*pb.RemoveNamespaceRoleResponse, error) {
	info := auth.MustFromContext(ctx)

	if len(info.NamespaceRoles) > 0 && !info.IsPlatformAdmin {
		if err := info.Require(auth.PermNamespaceMembershipManage, req.GetNamespace()); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	} else {
		if err := info.Require(auth.PermNamespaceMembershipManage, ""); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	if err := s.svc.RemoveNamespaceRole(ctx, info.TenantID, req.GetSubjectId(), req.GetNamespace()); err != nil {
		return nil, toGRPCError(ctx, err, "remove namespace role")
	}

	return &pb.RemoveNamespaceRoleResponse{}, nil
}

func (s *NamespaceServer) ListNamespaceMembers(ctx context.Context, req *pb.ListNamespaceMembersRequest) (*pb.ListNamespaceMembersResponse, error) {
	info := auth.MustFromContext(ctx)

	// Any user with namespace:read on this namespace can list members.
	// For users with memberships, check the specific namespace; otherwise tenant-level.
	if len(info.NamespaceRoles) > 0 && !info.IsPlatformAdmin {
		if err := info.Require(auth.PermNamespaceRead, req.GetNamespace()); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	} else {
		if err := info.Require(auth.PermNamespaceRead, ""); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	limit, offset := paginationDefaults(req.GetPagination())

	members, total, err := s.svc.ListNamespaceMembers(ctx, info.TenantID, req.GetNamespace(), int(limit), int(offset))
	if err != nil {
		return nil, toGRPCError(ctx, err, "list namespace members")
	}

	pbMembers := make([]*pb.NamespaceMembership, len(members))
	for i, m := range members {
		pbMembers[i] = membershipToProto(m)
	}

	return &pb.ListNamespaceMembersResponse{
		Members: pbMembers,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(total),
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

func (s *NamespaceServer) GetUserNamespaces(ctx context.Context, req *pb.GetUserNamespacesRequest) (*pb.GetUserNamespacesResponse, error) {
	info := auth.MustFromContext(ctx)

	subjectID := req.GetSubjectId()
	if subjectID == "" {
		// Default to caller's own namespaces
		subjectID = info.SubjectID
		if subjectID == "" {
			return nil, status.Error(codes.InvalidArgument, "subject_id is required (API keys have no subject)")
		}
	} else if subjectID != info.SubjectID {
		// Looking up another user's namespaces requires manage_members permission
		if err := info.Require(auth.PermNamespaceMembershipManage, ""); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	memberships, err := s.svc.GetUserNamespaces(ctx, info.TenantID, subjectID)
	if err != nil {
		return nil, toGRPCError(ctx, err, "get user namespaces")
	}

	pbMemberships := make([]*pb.NamespaceMembership, len(memberships))
	for i, m := range memberships {
		pbMemberships[i] = membershipToProto(m)
	}

	return &pb.GetUserNamespacesResponse{
		Memberships: pbMemberships,
	}, nil
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

func membershipToProto(m *namespace.Membership) *pb.NamespaceMembership {
	if m == nil {
		return nil
	}
	return &pb.NamespaceMembership{
		Id:        m.ID.String(),
		TenantId:  m.TenantID.String(),
		SubjectId: m.SubjectID,
		Namespace: m.Namespace,
		Role:      string(m.Role),
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
	}
}
