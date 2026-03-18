package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/audit"
	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/pkg/storage"
	pb "github.com/sarathsp06/sparrow/proto"
)

// TenantServer implements the TenantService and APIKeyService gRPC services.
type TenantServer struct {
	pb.UnimplementedTenantServiceServer
	pb.UnimplementedAPIKeyServiceServer
	svc   *tenant.Service
	audit *audit.Logger
}

var _ pb.TenantServiceServer = (*TenantServer)(nil)
var _ pb.APIKeyServiceServer = (*TenantServer)(nil)

// NewTenantServer creates a new TenantServer.
func NewTenantServer(svc *tenant.Service, auditLogger *audit.Logger) *TenantServer {
	return &TenantServer{svc: svc, audit: auditLogger}
}

// ---- TenantService ----

func (s *TenantServer) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.CreateTenantResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantCreate, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	var opts []tenant.CreateTenantOpts
	if req.ExternalId != nil {
		extID := req.GetExternalId()
		opts = append(opts, tenant.CreateTenantOpts{ExternalID: &extID})
	}

	t, err := s.svc.CreateTenant(ctx, req.GetName(), opts...)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "create tenant: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionTenantCreate,
		ResourceType: audit.ResourceTenant,
		ResourceID:   t.ID.String(),
		Metadata: map[string]any{
			"name": t.Name,
		},
	})

	return &pb.CreateTenantResponse{
		Tenant: tenantToProto(t),
	}, nil
}

func (s *TenantServer) GetTenant(ctx context.Context, req *pb.GetTenantRequest) (*pb.GetTenantResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantRead, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	// Non-platform-admins can only read their own tenant
	if !info.IsPlatformAdmin && id != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot access other tenants")
	}

	t, err := s.svc.GetTenant(ctx, id)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "get tenant: %v", err)
	}

	return &pb.GetTenantResponse{
		Tenant: tenantToProto(t),
	}, nil
}

func (s *TenantServer) ListTenants(ctx context.Context, req *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	info := auth.MustFromContext(ctx)

	// Only platform admins can list all tenants
	if !info.IsPlatformAdmin {
		return nil, status.Error(codes.PermissionDenied, "only platform admins can list tenants")
	}

	limit, offset := paginationDefaults(req.GetPagination())

	tenants, total, err := s.svc.ListTenants(ctx, int(limit), int(offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tenants: %v", err)
	}

	pbTenants := make([]*pb.Tenant, len(tenants))
	for i, t := range tenants {
		pbTenants[i] = tenantToProto(t)
	}

	return &pb.ListTenantsResponse{
		Tenants: pbTenants,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(total),
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

func (s *TenantServer) UpdateTenant(ctx context.Context, req *pb.UpdateTenantRequest) (*pb.UpdateTenantResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantUpdate, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	// Non-platform-admins can only update their own tenant
	if !info.IsPlatformAdmin && id != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot modify other tenants")
	}

	t, err := s.svc.UpdateTenant(ctx, id, req.GetName(), req.GetStatus())
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "update tenant: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionTenantUpdate,
		ResourceType: audit.ResourceTenant,
		ResourceID:   t.ID.String(),
		Metadata: map[string]any{
			"name":   req.GetName(),
			"status": req.GetStatus(),
		},
	})

	return &pb.UpdateTenantResponse{
		Tenant: tenantToProto(t),
	}, nil
}

func (s *TenantServer) DeleteTenant(ctx context.Context, req *pb.DeleteTenantRequest) (*pb.DeleteTenantResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantDelete, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	// Only platform admins can delete tenants
	if !info.IsPlatformAdmin {
		return nil, status.Error(codes.PermissionDenied, "only platform admins can delete tenants")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	if err := s.svc.DeleteTenant(ctx, id); err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "delete tenant: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionTenantDelete,
		ResourceType: audit.ResourceTenant,
		ResourceID:   id.String(),
	})

	return &pb.DeleteTenantResponse{}, nil
}

// ---- APIKeyService ----

func (s *TenantServer) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageAPIKeys, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	// Non-platform-admins can only create keys for their own tenant
	if !info.IsPlatformAdmin && tenantID != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot create API keys for other tenants")
	}

	createReq := tenant.CreateAPIKeyRequest{
		TenantID:        tenantID,
		Name:            req.GetName(),
		Role:            auth.Role(req.GetRole()),
		IsPlatformAdmin: req.GetIsPlatformAdmin(),
	}
	if req.NamespaceScope != nil {
		ns := req.GetNamespaceScope()
		createReq.NamespaceScope = &ns
	}
	if req.ExpiresAt != nil {
		t := req.GetExpiresAt().AsTime()
		createReq.ExpiresAt = &t
	}

	// Non-platform-admins cannot grant platform admin
	if createReq.IsPlatformAdmin && !info.IsPlatformAdmin {
		return nil, status.Error(codes.PermissionDenied, "only platform admins can create platform admin keys")
	}

	result, err := s.svc.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create API key: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionAPIKeyCreate,
		ResourceType: audit.ResourceAPIKey,
		ResourceID:   result.Key.ID.String(),
		Metadata: map[string]any{
			"name":      req.GetName(),
			"tenant_id": tenantID.String(),
			"role":      req.GetRole(),
		},
	})

	return &pb.CreateAPIKeyResponse{
		Key:    apiKeyToProto(result.Key),
		RawKey: result.RawKey,
	}, nil
}

func (s *TenantServer) GetAPIKey(ctx context.Context, req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageAPIKeys, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid API key ID: %v", err)
	}

	key, err := s.svc.GetAPIKey(ctx, id)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "API key not found")
		}
		return nil, status.Errorf(codes.Internal, "get API key: %v", err)
	}

	// Non-platform-admins can only see keys for their own tenant
	if !info.IsPlatformAdmin && key.TenantID != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot access API keys for other tenants")
	}

	return &pb.GetAPIKeyResponse{
		Key: apiKeyToProto(key),
	}, nil
}

func (s *TenantServer) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageAPIKeys, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	// Non-platform-admins can only list keys for their own tenant
	if !info.IsPlatformAdmin && tenantID != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot list API keys for other tenants")
	}

	limit, offset := paginationDefaults(req.GetPagination())

	keys, total, err := s.svc.ListAPIKeys(ctx, tenantID, int(limit), int(offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list API keys: %v", err)
	}

	pbKeys := make([]*pb.APIKey, len(keys))
	for i, k := range keys {
		pbKeys[i] = apiKeyToProto(k)
	}

	return &pb.ListAPIKeysResponse{
		Keys: pbKeys,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(total),
			Limit:      limit,
			Offset:     offset,
		},
	}, nil
}

func (s *TenantServer) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageAPIKeys, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid API key ID: %v", err)
	}

	// Look up the key to verify tenant ownership
	key, err := s.svc.GetAPIKey(ctx, id)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "API key not found")
		}
		return nil, status.Errorf(codes.Internal, "get API key: %v", err)
	}

	if !info.IsPlatformAdmin && key.TenantID != info.TenantID {
		return nil, status.Error(codes.PermissionDenied, "cannot revoke API keys for other tenants")
	}

	if err := s.svc.RevokeAPIKey(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "revoke API key: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionAPIKeyRevoke,
		ResourceType: audit.ResourceAPIKey,
		ResourceID:   id.String(),
		Metadata: map[string]any{
			"tenant_id": key.TenantID.String(),
		},
	})

	return &pb.RevokeAPIKeyResponse{}, nil
}

// ---- Conversion helpers ----

func tenantToProto(t *tenant.Tenant) *pb.Tenant {
	if t == nil {
		return nil
	}
	p := &pb.Tenant{
		Id:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		Status:    t.Status,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}
	if t.ExternalID != nil {
		p.ExternalId = t.ExternalID
	}
	return p
}

func apiKeyToProto(k *tenant.APIKey) *pb.APIKey {
	if k == nil {
		return nil
	}
	pbKey := &pb.APIKey{
		Id:              k.ID.String(),
		TenantId:        k.TenantID.String(),
		Name:            k.Name,
		KeyPrefix:       k.KeyPrefix,
		Role:            string(k.Role),
		IsPlatformAdmin: k.IsPlatformAdmin,
		CreatedAt:       timestamppb.New(k.CreatedAt),
	}
	if k.NamespaceScope != nil {
		pbKey.NamespaceScope = k.NamespaceScope
	}
	if k.ExpiresAt != nil {
		pbKey.ExpiresAt = timestamppb.New(*k.ExpiresAt)
	}
	if k.LastUsedAt != nil {
		pbKey.LastUsedAt = timestamppb.New(*k.LastUsedAt)
	}
	if k.RevokedAt != nil {
		pbKey.RevokedAt = timestamppb.New(*k.RevokedAt)
	}
	return pbKey
}

func paginationDefaults(p *pb.PaginationRequest) (limit, offset int32) {
	limit = 20
	offset = 0
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if p.Offset > 0 {
			offset = p.Offset
		}
	}
	if limit > 100 {
		limit = 100
	}
	return limit, offset
}
