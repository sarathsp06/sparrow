package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

// TenantConnectServer implements the Connect-RPC handlers for TenantService
// and APIKeyService by delegating to the gRPC server implementations.
type TenantConnectServer struct {
	tenantService pb.TenantServiceServer
	apiKeyService pb.APIKeyServiceServer
}

// NewTenantConnectServer creates a new Connect-RPC adapter for tenant and API key services.
func NewTenantConnectServer(
	tenantService pb.TenantServiceServer,
	apiKeyService pb.APIKeyServiceServer,
) *TenantConnectServer {
	return &TenantConnectServer{
		tenantService: tenantService,
		apiKeyService: apiKeyService,
	}
}

// ---- TenantService ----

func (s *TenantConnectServer) CreateTenant(ctx context.Context, req *connect.Request[pb.CreateTenantRequest]) (*connect.Response[pb.CreateTenantResponse], error) {
	res, err := s.tenantService.CreateTenant(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) GetTenant(ctx context.Context, req *connect.Request[pb.GetTenantRequest]) (*connect.Response[pb.GetTenantResponse], error) {
	res, err := s.tenantService.GetTenant(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) ListTenants(ctx context.Context, req *connect.Request[pb.ListTenantsRequest]) (*connect.Response[pb.ListTenantsResponse], error) {
	res, err := s.tenantService.ListTenants(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) UpdateTenant(ctx context.Context, req *connect.Request[pb.UpdateTenantRequest]) (*connect.Response[pb.UpdateTenantResponse], error) {
	res, err := s.tenantService.UpdateTenant(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) DeleteTenant(ctx context.Context, req *connect.Request[pb.DeleteTenantRequest]) (*connect.Response[pb.DeleteTenantResponse], error) {
	res, err := s.tenantService.DeleteTenant(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// ---- APIKeyService ----

func (s *TenantConnectServer) CreateAPIKey(ctx context.Context, req *connect.Request[pb.CreateAPIKeyRequest]) (*connect.Response[pb.CreateAPIKeyResponse], error) {
	res, err := s.apiKeyService.CreateAPIKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) GetAPIKey(ctx context.Context, req *connect.Request[pb.GetAPIKeyRequest]) (*connect.Response[pb.GetAPIKeyResponse], error) {
	res, err := s.apiKeyService.GetAPIKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) ListAPIKeys(ctx context.Context, req *connect.Request[pb.ListAPIKeysRequest]) (*connect.Response[pb.ListAPIKeysResponse], error) {
	res, err := s.apiKeyService.ListAPIKeys(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TenantConnectServer) RevokeAPIKey(ctx context.Context, req *connect.Request[pb.RevokeAPIKeyRequest]) (*connect.Response[pb.RevokeAPIKeyResponse], error) {
	res, err := s.apiKeyService.RevokeAPIKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// Compile-time interface checks
var _ pbconnect.TenantServiceHandler = (*TenantConnectServer)(nil)
var _ pbconnect.APIKeyServiceHandler = (*TenantConnectServer)(nil)
