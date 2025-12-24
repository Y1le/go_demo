package websocket

import (
	"context"
	"encoding/json"
	client "liam/internal/client"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境需要更严格的检查
	},
}

type WSHandler struct {
	grpcClient *client.WerewolfGRPCClient
	clients    map[string]*WSClient
	mu         sync.RWMutex
}

type WSClient struct {
	conn     *websocket.Conn
	roomID   string
	playerID string
	send     chan []byte
}

type WSMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func NewWSHandler(grpcClient *client.WerewolfGRPCClient) *WSHandler {
	return &WSHandler{
		grpcClient: grpcClient,
		clients:    make(map[string]*WSClient),
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	roomID := c.Query("room_id")
	playerID := c.Query("player_id")

	if roomID == "" || playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id and player_id required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &WSClient{
		conn:     conn,
		roomID:   roomID,
		playerID: playerID,
		send:     make(chan []byte, 256),
	}

	h.mu.Lock()
	h.clients[playerID] = client
	h.mu.Unlock()

	// 启动读写协程
	go h.readPump(client)
	go h.writePump(client)
	go h.subscribeGameEvents(client)
}

func (h *WSHandler) readPump(client *WSClient) {
	defer func() {
		h.removeClient(client.playerID)
		client.conn.Close()
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
		h.handleClientMessage(client, message)
	}
}

func (h *WSHandler) writePump(client *WSClient) {
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

func (h *WSHandler) subscribeGameEvents(client *WSClient) {
	ctx := context.Background()
	stream, err := h.grpcClient.SubscribeGameEvents(ctx, client.roomID, client.playerID)
	if err != nil {
		log.Printf("Failed to subscribe to game events: %v", err)
		return
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving event: %v", err)
			return
		}

		// 将事件发送给 WebSocket 客户端
		eventData := map[string]interface{}{
			"type":       event.EventType,
			"message":    event.Message,
			"phase_info": event.PhaseInfo,
			"extra_data": event.ExtraData,
			"timestamp":  event.Timestamp,
		}
		/*
			state           protoimpl.MessageState `protogen:"open.v1"`
			EventType       GameEvent_EventType    `protobuf:"varint,1,opt,name=event_type,json=eventType,proto3,enum=werewolf.GameEvent_EventType" json:"event_type,omitempty"`
			Message         string                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
			PhaseInfo       *PhaseInfo             `protobuf:"bytes,3,opt,name=phase_info,json=phaseInfo,proto3" json:"phase_info,omitempty"`
			AffectedPlayers []*Player              `protobuf:"bytes,4,rep,name=affected_players,json=affectedPlayers,proto3" json:"affected_players,omitempty"`
			Timestamp       int64                  `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
			ExtraData       map[string]string      `protobuf:"bytes,6,rep,name=extra_data,json=extraData,proto3" json:"extra_data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
			unknownFields   protoimpl.UnknownFields
			sizeCache       protoimpl.SizeCache
		*/
		msg := WSMessage{
			Type:    "game_event",
			Payload: eventData,
		}

		data, _ := json.Marshal(msg)
		client.send <- data
	}
}

func (h *WSHandler) handleClientMessage(client *WSClient, message []byte) {
	var msg WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// 根据消息类型处理
	switch msg.Type {
	case "ping":
		response := WSMessage{
			Type:    "pong",
			Payload: map[string]interface{}{"timestamp": time.Now().Unix()},
		}
		data, _ := json.Marshal(response)
		client.send <- data
	}
}

func (h *WSHandler) removeClient(playerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client, ok := h.clients[playerID]; ok {
		close(client.send)
		delete(h.clients, playerID)
	}
}

// BroadcastToRoom 向房间内所有客户端广播消息
func (h *WSHandler) BroadcastToRoom(roomID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.roomID == roomID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client.playerID)
			}
		}
	}
}
