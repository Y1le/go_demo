package client

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	pb "liam/pkg/werewolf"

	"github.com/gorilla/websocket"
)

// WSManager WebSocket 连接管理器
type WSManager struct {
	grpcClient *WerewolfGRPCClient
	clients    map[string]*WSClient
	mu         sync.RWMutex
}

// WSClient WebSocket 客户端
type WSClient struct {
	conn         *websocket.Conn
	roomID       string
	playerID     string
	send         chan []byte
	grpcStream   pb.WerewolfService_SubscribeGameEventsClient
	cancelStream context.CancelFunc
	mu           sync.Mutex
}

// WSMessage WebSocket 消息格式
type WSMessage struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GameEventMessage 游戏事件消息
type GameEventMessage struct {
	EventType       string            `json:"event_type"`
	Message         string            `json:"message"`
	PhaseInfo       *PhaseInfo        `json:"phase_info,omitempty"`
	AffectedPlayers []PlayerInfo      `json:"affected_players,omitempty"`
	ExtraData       map[string]string `json:"extra_data,omitempty"`
}

type PhaseInfo struct {
	CurrentPhase string `json:"current_phase"`
	PhaseName    string `json:"phase_name"`
	TimeLimit    int32  `json:"time_limit"`
}

type PlayerInfo struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	Position int32  `json:"position"`
	IsAlive  bool   `json:"is_alive"`
}

func NewWSManager(grpcClient *WerewolfGRPCClient) *WSManager {
	return &WSManager{
		grpcClient: grpcClient,
		clients:    make(map[string]*WSClient),
	}
}

// RegisterClient 注册新的 WebSocket 客户端
func (m *WSManager) RegisterClient(conn *websocket.Conn, roomID, playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已经存在
	if existingClient, exists := m.clients[playerID]; exists {
		existingClient.Close()
	}

	client := &WSClient{
		conn:     conn,
		roomID:   roomID,
		playerID: playerID,
		send:     make(chan []byte, 256),
	}

	m.clients[playerID] = client

	// 启动读写协程
	go m.readPump(client)
	go m.writePump(client)
	go m.subscribeGameEvents(client)

	log.Printf("玩家 %s 连接到房间 %s", playerID, roomID)

	return nil
}

// UnregisterClient 注销客户端
func (m *WSManager) UnregisterClient(playerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[playerID]; exists {
		client.Close()
		delete(m.clients, playerID)
		log.Printf("玩家 %s 断开连接", playerID)
	}
}

// readPump 读取客户端消息
func (m *WSManager) readPump(client *WSClient) {
	defer func() {
		m.UnregisterClient(client.playerID)
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 处理客户端消息
		m.handleClientMessage(client, message)
	}
}

// writePump 向客户端写入消息
func (m *WSManager) writePump(client *WSClient) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("写入消息失败: %v", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// subscribeGameEvents 订阅游戏事件
func (m *WSManager) subscribeGameEvents(client *WSClient) {
	ctx, cancel := context.WithCancel(context.Background())
	client.cancelStream = cancel

	stream, err := m.grpcClient.SubscribeGameEvents(ctx, client.roomID, client.playerID)
	if err != nil {
		log.Printf("订阅游戏事件失败: %v", err)
		return
	}

	client.grpcStream = stream

	for {
		event, err := stream.Recv()
		if err != nil {
			log.Printf("接收事件失败: %v", err)
			return
		}

		// 转换为 WebSocket 消息
		message := m.convertEventToWSMessage(event)
		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("序列化消息失败: %v", err)
			continue
		}

		select {
		case client.send <- data:
		default:
			log.Printf("客户端 %s 消息队列已满", client.playerID)
		}
	}
}

// convertEventToWSMessage 转换 gRPC 事件为 WebSocket 消息
func (m *WSManager) convertEventToWSMessage(event *pb.GameEvent) *WSMessage {
	eventData := GameEventMessage{
		EventType: event.EventType.String(),
		Message:   event.Message,
		ExtraData: event.ExtraData,
	}

	// 转换阶段信息
	if event.PhaseInfo != nil {
		eventData.PhaseInfo = &PhaseInfo{
			CurrentPhase: event.PhaseInfo.CurrentPhase.String(),
			PhaseName:    event.PhaseInfo.PhaseName,
			TimeLimit:    event.PhaseInfo.TimeLimit,
		}
	}

	// 转换受影响的玩家
	if len(event.AffectedPlayers) > 0 {
		eventData.AffectedPlayers = make([]PlayerInfo, len(event.AffectedPlayers))
		for i, p := range event.AffectedPlayers {
			eventData.AffectedPlayers[i] = PlayerInfo{
				PlayerID: p.PlayerId,
				Name:     p.Name,
				Position: p.Position,
				IsAlive:  p.IsAlive,
			}
		}
	}

	return &WSMessage{
		Type:      "game_event",
		Timestamp: event.Timestamp,
		Data: map[string]interface{}{
			"event": eventData,
		},
	}
}

// handleClientMessage 处理客户端消息
func (m *WSManager) handleClientMessage(client *WSClient, message []byte) {
	var msg WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	switch msg.Type {
	case "ping":
		// 响应 ping
		response := WSMessage{
			Type:      "pong",
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(response)
		client.send <- data

	case "heartbeat":
		// 心跳响应
		response := WSMessage{
			Type:      "heartbeat_ack",
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(response)
		client.send <- data

	default:
		log.Printf("未知消息类型: %s", msg.Type)
	}
}

// BroadcastToRoom 向房间广播消息
func (m *WSManager) BroadcastToRoom(roomID string, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		if client.roomID == roomID {
			select {
			case client.send <- message:
			default:
				log.Printf("客户端 %s 消息队列已满，跳过广播", client.playerID)
			}
		}
	}
}

// SendToPlayer 向特定玩家发送消息
func (m *WSManager) SendToPlayer(playerID string, message []byte) error {
	m.mu.RLock()
	client, exists := m.clients[playerID]
	m.mu.RUnlock()

	if !exists {
		return nil // 玩家不在线，忽略
	}

	select {
	case client.send <- message:
		return nil
	default:
		return nil // 队列满了，忽略
	}
}

// Close 关闭 WebSocket 客户端
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelStream != nil {
		c.cancelStream()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	if c.send != nil {
		close(c.send)
	}
}

// GetOnlinePlayerCount 获取在线玩家数量
func (m *WSManager) GetOnlinePlayerCount(roomID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, client := range m.clients {
		if client.roomID == roomID {
			count++
		}
	}
	return count
}
