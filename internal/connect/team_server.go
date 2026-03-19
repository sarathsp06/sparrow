package connect

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

// TeamConnectServer implements the Connect-RPC handlers for TeamService
// by delegating to the gRPC server implementation.
type TeamConnectServer struct {
	teamService pb.TeamServiceServer
}

// NewTeamConnectServer creates a new Connect-RPC adapter for the team service.
func NewTeamConnectServer(teamService pb.TeamServiceServer) *TeamConnectServer {
	return &TeamConnectServer{teamService: teamService}
}

func (s *TeamConnectServer) ListMembers(ctx context.Context, req *connect.Request[pb.ListMembersRequest]) (*connect.Response[pb.ListMembersResponse], error) {
	res, err := s.teamService.ListMembers(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TeamConnectServer) InviteMember(ctx context.Context, req *connect.Request[pb.InviteMemberRequest]) (*connect.Response[pb.InviteMemberResponse], error) {
	res, err := s.teamService.InviteMember(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TeamConnectServer) RemoveMember(ctx context.Context, req *connect.Request[pb.RemoveMemberRequest]) (*connect.Response[pb.RemoveMemberResponse], error) {
	res, err := s.teamService.RemoveMember(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TeamConnectServer) UpdateMemberRole(ctx context.Context, req *connect.Request[pb.UpdateMemberRoleRequest]) (*connect.Response[pb.UpdateMemberRoleResponse], error) {
	res, err := s.teamService.UpdateMemberRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TeamConnectServer) ListInvitations(ctx context.Context, req *connect.Request[pb.ListInvitationsRequest]) (*connect.Response[pb.ListInvitationsResponse], error) {
	res, err := s.teamService.ListInvitations(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (s *TeamConnectServer) RevokeInvitation(ctx context.Context, req *connect.Request[pb.RevokeInvitationRequest]) (*connect.Response[pb.RevokeInvitationResponse], error) {
	res, err := s.teamService.RevokeInvitation(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// Compile-time interface check
var _ pbconnect.TeamServiceHandler = (*TeamConnectServer)(nil)
