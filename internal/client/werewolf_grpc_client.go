package client

import (
	"context"
	"fmt"
	"time"

	pb "liam/pkg/werewolf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WerewolfGRPCClient struct {
	conn   *grpc.ClientConn
	client pb.WerewolfServiceClient
}

func NewWerewolfGRPCClient(address string) (*WerewolfGRPCClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("连接 gRPC 服务失败: %v", err)
	}

	return &WerewolfGRPCClient{
		conn:   conn,
		client: pb.NewWerewolfServiceClient(conn),
	}, nil
}

func (c *WerewolfGRPCClient) Close() error {
	return c.conn.Close()
}

// CreateRoom 创建游戏房间
func (c *WerewolfGRPCClient) CreateRoom(ctx context.Context, roomName string, maxPlayers int32, roleConfig map[string]int32) (*pb.CreateRoomResponse, error) {
	return c.client.CreateRoom(ctx, &pb.CreateRoomRequest{
		RoomName:   roomName,
		MaxPlayers: maxPlayers,
		RoleConfig: roleConfig,
	})
}

// JoinRoom 加入房间
func (c *WerewolfGRPCClient) JoinRoom(ctx context.Context, roomID, playerID, playerName string) (*pb.JoinRoomResponse, error) {
	return c.client.JoinRoom(ctx, &pb.JoinRoomRequest{
		RoomId:     roomID,
		PlayerId:   playerID,
		PlayerName: playerName,
	})
}

// StartGame 开始游戏
func (c *WerewolfGRPCClient) StartGame(ctx context.Context, roomID string) (*pb.StartGameResponse, error) {
	return c.client.StartGame(ctx, &pb.StartGameRequest{
		RoomId: roomID,
	})
}

// NightAction 夜晚行动
func (c *WerewolfGRPCClient) NightAction(ctx context.Context, roomID, playerID, targetID, actionType string) (*pb.NightActionResponse, error) {
	return c.client.NightAction(ctx, &pb.NightActionRequest{
		RoomId:         roomID,
		PlayerId:       playerID,
		TargetPlayerId: targetID,
		ActionType:     actionType,
	})
}

// Vote 投票
func (c *WerewolfGRPCClient) Vote(ctx context.Context, roomID, voterID, targetID string) (*pb.VoteResponse, error) {
	return c.client.Vote(ctx, &pb.VoteRequest{
		RoomId:   roomID,
		VoterId:  voterID,
		TargetId: targetID,
	})
}

// GetGameState 获取游戏状态
func (c *WerewolfGRPCClient) GetGameState(ctx context.Context, roomID, playerID string) (*pb.GetGameStateResponse, error) {
	return c.client.GetGameState(ctx, &pb.GetGameStateRequest{
		RoomId:   roomID,
		PlayerId: playerID,
	})
}

// SubscribeGameEvents 订阅游戏事件
func (c *WerewolfGRPCClient) SubscribeGameEvents(ctx context.Context, roomID, playerID string) (pb.WerewolfService_SubscribeGameEventsClient, error) {
	return c.client.SubscribeGameEvents(ctx, &pb.GetGameStateRequest{
		RoomId:   roomID,
		PlayerId: playerID,
	})
}
