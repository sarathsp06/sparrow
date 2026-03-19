package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sarathsp06/sparrow/internal/audit"
	"github.com/sarathsp06/sparrow/internal/auth"
	pb "github.com/sarathsp06/sparrow/proto"
)

// TeamServer implements the TeamService gRPC service.
// It delegates to the configured IdentityProvider's TeamManager for all
// org-level member and invitation operations.
type TeamServer struct {
	pb.UnimplementedTeamServiceServer
	provider     auth.IdentityProvider
	tenantLookup auth.ExternalTenantLookup
	audit        *audit.Logger
}

var _ pb.TeamServiceServer = (*TeamServer)(nil)

// NewTeamServer creates a new TeamServer.
// provider is the identity provider (Clerk, Noop, etc.).
// tenantLookup resolves internal tenant UUIDs to external org IDs.
// Either may be nil if the deployment doesn't support team management.
func NewTeamServer(provider auth.IdentityProvider, tenantLookup auth.ExternalTenantLookup, auditLogger *audit.Logger) *TeamServer {
	return &TeamServer{
		provider:     provider,
		tenantLookup: tenantLookup,
		audit:        auditLogger,
	}
}

// teamManager returns the TeamManager or an Unimplemented error if not supported.
func (s *TeamServer) teamManager() (auth.TeamManager, error) {
	if s.provider == nil {
		return nil, status.Error(codes.Unimplemented, "team management is not available: no identity provider configured")
	}
	tm := s.provider.TeamManagement()
	if tm == nil {
		return nil, status.Error(codes.Unimplemented, "team management is not available: identity provider does not support it")
	}
	return tm, nil
}

// resolveExternalOrgID maps the caller's internal tenant UUID to the identity provider's org ID.
func (s *TeamServer) resolveExternalOrgID(ctx context.Context, info *auth.AuthInfo) (string, error) {
	if s.tenantLookup == nil {
		return "", status.Error(codes.FailedPrecondition, "external tenant lookup not configured")
	}
	externalID, err := s.tenantLookup.LookupExternalIDByTenantID(ctx, info.TenantID)
	if err != nil {
		return "", status.Errorf(codes.Internal, "resolve external org ID: %v", err)
	}
	return externalID, nil
}

func (s *TeamServer) ListMembers(ctx context.Context, req *pb.ListMembersRequest) (*pb.ListMembersResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	limit, offset := paginationDefaults(req.GetPagination())
	members, total, err := tm.ListMembers(ctx, externalOrgID, int(limit), int(offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list members: %v", err)
	}

	pbMembers := make([]*pb.TeamMember, len(members))
	for i, m := range members {
		pbMembers[i] = teamMemberToProto(m)
	}

	return &pb.ListMembersResponse{
		Members:    pbMembers,
		TotalCount: int32(total),
	}, nil
}

func (s *TeamServer) InviteMember(ctx context.Context, req *pb.InviteMemberRequest) (*pb.InviteMemberResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetRole() == "" {
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	invitation, err := tm.InviteMember(ctx, externalOrgID, req.GetEmail(), req.GetRole())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invite member: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionMemberInvite,
		ResourceType: audit.ResourceMember,
		ResourceID:   invitation.ID,
		Metadata: map[string]any{
			"email": req.GetEmail(),
			"role":  req.GetRole(),
		},
	})

	return &pb.InviteMemberResponse{
		Invitation: teamInvitationToProto(*invitation),
	}, nil
}

func (s *TeamServer) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	if err := tm.RemoveMember(ctx, externalOrgID, req.GetUserId()); err != nil {
		return nil, status.Errorf(codes.Internal, "remove member: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionMemberRemove,
		ResourceType: audit.ResourceMember,
		ResourceID:   req.GetUserId(),
	})

	return &pb.RemoveMemberResponse{}, nil
}

func (s *TeamServer) UpdateMemberRole(ctx context.Context, req *pb.UpdateMemberRoleRequest) (*pb.UpdateMemberRoleResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetRole() == "" {
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	member, err := tm.UpdateMemberRole(ctx, externalOrgID, req.GetUserId(), req.GetRole())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update member role: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionMemberUpdateRole,
		ResourceType: audit.ResourceMember,
		ResourceID:   req.GetUserId(),
		Metadata: map[string]any{
			"role": req.GetRole(),
		},
	})

	return &pb.UpdateMemberRoleResponse{
		Member: teamMemberToProto(*member),
	}, nil
}

func (s *TeamServer) ListInvitations(ctx context.Context, req *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	limit, offset := paginationDefaults(req.GetPagination())
	invitations, total, err := tm.ListInvitations(ctx, externalOrgID, req.GetStatus(), int(limit), int(offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list invitations: %v", err)
	}

	pbInvitations := make([]*pb.TeamInvitation, len(invitations))
	for i, inv := range invitations {
		pbInvitations[i] = teamInvitationToProto(inv)
	}

	return &pb.ListInvitationsResponse{
		Invitations: pbInvitations,
		TotalCount:  int32(total),
	}, nil
}

func (s *TeamServer) RevokeInvitation(ctx context.Context, req *pb.RevokeInvitationRequest) (*pb.RevokeInvitationResponse, error) {
	info := auth.MustFromContext(ctx)
	if err := info.Require(auth.PermTenantManageMembers, ""); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.GetInvitationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invitation_id is required")
	}

	tm, err := s.teamManager()
	if err != nil {
		return nil, err
	}

	externalOrgID, err := s.resolveExternalOrgID(ctx, info)
	if err != nil {
		return nil, err
	}

	if err := tm.RevokeInvitation(ctx, externalOrgID, req.GetInvitationId()); err != nil {
		return nil, status.Errorf(codes.Internal, "revoke invitation: %v", err)
	}

	s.audit.Log(ctx, audit.LogEntry{
		Action:       audit.ActionMemberInviteRevoke,
		ResourceType: audit.ResourceMember,
		ResourceID:   req.GetInvitationId(),
	})

	return &pb.RevokeInvitationResponse{}, nil
}

// ---- Conversion helpers ----

func teamMemberToProto(m auth.TeamMember) *pb.TeamMember {
	return &pb.TeamMember{
		UserId:    m.UserID,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Email:     m.Email,
		ImageUrl:  m.ImageURL,
		Role:      m.Role,
		JoinedAt:  timestamppb.New(m.JoinedAt),
	}
}

func teamInvitationToProto(inv auth.TeamInvitation) *pb.TeamInvitation {
	pbInv := &pb.TeamInvitation{
		Id:        inv.ID,
		Email:     inv.Email,
		Role:      inv.Role,
		Status:    inv.Status,
		CreatedAt: timestamppb.New(inv.CreatedAt),
	}
	if !inv.ExpiresAt.IsZero() {
		pbInv.ExpiresAt = timestamppb.New(inv.ExpiresAt)
	}
	return pbInv
}
