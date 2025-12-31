package services

import (
	"context"
	"fmt"
	"liam/internal/client"
	dto "liam/internal/dto/werewolf"
)

type WerewolfService struct {
	grpcClient *client.WerewolfGRPCClient
}

func NewWerewolfService(grpcClient *client.WerewolfGRPCClient) *WerewolfService {
	return &WerewolfService{
		grpcClient: grpcClient,
	}
}

// CreateRoom 创建房间
func (s *WerewolfService) CreateRoom(ctx context.Context, req *dto.CreateRoomRequest) (*dto.CreateRoomResponse, error) {
	// 验证角色配置
	if err := s.validateRoleConfig(req.RoleConfig, req.MaxPlayers); err != nil {
		return nil, err
	}

	roleConfig := make(map[string]int32)
	for k, v := range req.RoleConfig {
		roleConfig[k] = int32(v)
	}

	resp, err := s.grpcClient.CreateRoom(ctx, req.RoomName, int32(req.MaxPlayers), roleConfig)
	if err != nil {
		return nil, err
	}

	return &dto.CreateRoomResponse{
		RoomID:  resp.RoomId,
		Message: resp.Message,
	}, nil
}

// JoinRoom 加入房间
func (s *WerewolfService) JoinRoom(ctx context.Context, req *dto.JoinRoomRequest) (*dto.JoinRoomResponse, error) {
	resp, err := s.grpcClient.JoinRoom(ctx, req.RoomID, req.PlayerID, req.PlayerName)
	if err != nil {
		return nil, err
	}

	return &dto.JoinRoomResponse{
		Success:  resp.Success,
		Message:  resp.Message,
		PlayerID: req.PlayerID,
		Position: resp.Player.Position,
	}, nil
}

// StartGame 开始游戏
func (s *WerewolfService) StartGame(ctx context.Context, req *dto.StartGameRequest) (*dto.StartGameResponse, error) {
	resp, err := s.grpcClient.StartGame(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	var phaseInfo *dto.PhaseInfo
	if resp.PhaseInfo != nil {
		phaseInfo = &dto.PhaseInfo{
			CurrentPhase: resp.PhaseInfo.CurrentPhase.String(),
			PhaseName:    resp.PhaseInfo.PhaseName,
			TimeLimit:    resp.PhaseInfo.TimeLimit,
		}
	}

	return &dto.StartGameResponse{
		Success:   resp.Success,
		Message:   resp.Message,
		PhaseInfo: phaseInfo,
	}, nil
}

// NightAction 夜晚行动
func (s *WerewolfService) NightAction(ctx context.Context, req *dto.NightActionRequest) (*dto.NightActionResponse, error) {
	resp, err := s.grpcClient.NightAction(ctx, req.RoomID, req.PlayerID, req.TargetID, req.ActionType)
	if err != nil {
		return nil, err
	}

	return &dto.NightActionResponse{
		Success: resp.Success,
		Message: resp.Message,
		Result:  resp.Result,
	}, nil
}

// Vote 投票
func (s *WerewolfService) Vote(ctx context.Context, req *dto.VoteRequest) (*dto.VoteResponse, error) {
	resp, err := s.grpcClient.Vote(ctx, req.RoomID, req.VoterID, req.TargetID)
	if err != nil {
		return nil, err
	}

	return &dto.VoteResponse{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// GetGameState 获取游戏状态
func (s *WerewolfService) GetGameState(ctx context.Context, roomID, playerID string) (*dto.GetGameStateResponse, error) {
	resp, err := s.grpcClient.GetGameState(ctx, roomID, playerID)
	if err != nil {
		return nil, err
	}
	players := make([]dto.PlayerInfo, len(resp.Players))
	for i, p := range resp.Players {
		players[i] = dto.PlayerInfo{
			PlayerID: p.PlayerId,
			Name:     p.Name,
			Role:     p.Role.String(),
			Camp:     p.Camp.String(),
			IsAlive:  p.IsAlive,
			Position: p.Position,
			CanAct:   p.CanAct,
		}
	}

	var currentPlayer *dto.PlayerInfo
	if resp.CurrentPlayer != nil {
		currentPlayer = &dto.PlayerInfo{
			PlayerID: resp.CurrentPlayer.PlayerId,
			Name:     resp.CurrentPlayer.Name,
			Role:     resp.CurrentPlayer.Role.String(),
			Camp:     resp.CurrentPlayer.Camp.String(),
			IsAlive:  resp.CurrentPlayer.IsAlive,
			Position: resp.CurrentPlayer.Position,
			CanAct:   resp.CurrentPlayer.CanAct,
		}
	}

	var phaseInfo *dto.PhaseInfo
	if resp.PhaseInfo != nil {
		phaseInfo = &dto.PhaseInfo{
			CurrentPhase: resp.PhaseInfo.CurrentPhase.String(),
			PhaseName:    resp.PhaseInfo.PhaseName,
			TimeLimit:    resp.PhaseInfo.TimeLimit,
		}
	}

	return &dto.GetGameStateResponse{
		RoomID:        resp.RoomId,
		State:         resp.State.String(),
		PhaseInfo:     phaseInfo,
		Players:       players,
		DayCount:      int(resp.DayCount),
		CurrentPlayer: currentPlayer,
	}, nil
}

// validateRoleConfig 验证角色配置
func (s *WerewolfService) validateRoleConfig(config map[string]int, maxPlayers int) error {
	totalPlayers := 0
	for _, count := range config {
		totalPlayers += count
	}
	if totalPlayers != maxPlayers {
		return fmt.Errorf("角色数量(%d)与最大玩家数(%d)不匹配", totalPlayers, maxPlayers)
	}

	// 至少要有1个狼人
	if config["werewolf"] < 1 {
		return fmt.Errorf("至少需要1个狼人")
	}

	// 至少要有1个村民
	if config["villager"] < 1 {
		return fmt.Errorf("至少需要1个村民")
	}

	return nil
}
