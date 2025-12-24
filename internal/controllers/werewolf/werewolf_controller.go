package controller

import (
	"liam/internal/client"
	dto "liam/internal/dto/werewolf"
	service "liam/internal/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境需要更严格的检查
		return true
	},
}

type WerewolfController struct {
	service   *service.WerewolfService
	wsManager *client.WSManager
}

func NewWerewolfController(service *service.WerewolfService, wsManager *client.WSManager) *WerewolfController {
	return &WerewolfController{
		service:   service,
		wsManager: wsManager,
	}
}

// CreateRoom 创建房间
// @Summary 创建游戏房间
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.CreateRoomRequest true "创建房间请求"
// @Success 200 {object} dto.CreateRoomResponse
// @Router /api/v1/rooms [post]
func (ctrl *WerewolfController) CreateRoom(c *gin.Context) {
	var req dto.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	resp, err := ctrl.service.CreateRoom(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// JoinRoom 加入房间
// @Summary 加入游戏房间
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.JoinRoomRequest true "加入房间请求"
// @Success 200 {object} dto.JoinRoomResponse
// @Router /api/v1/rooms/join [post]
func (ctrl *WerewolfController) JoinRoom(c *gin.Context) {
	var req dto.JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 生成玩家ID
	playerID := uuid.New().String()
	req.PlayerID = playerID

	resp, err := ctrl.service.JoinRoom(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	// 设置 Cookie 保存玩家ID
	c.SetCookie("player_id", playerID, 3600*24, "/", "", false, true)
	c.SetCookie("room_id", req.RoomID, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, resp)
}

// StartGame 开始游戏
// @Summary 开始游戏
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.StartGameRequest true "开始游戏请求"
// @Success 200 {object} dto.StartGameResponse
// @Router /api/v1/rooms/start [post]
func (ctrl *WerewolfController) StartGame(c *gin.Context) {
	var req dto.StartGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	resp, err := ctrl.service.StartGame(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// NightAction 夜晚行动
// @Summary 夜晚行动
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.NightActionRequest true "夜晚行动请求"
// @Success 200 {object} dto.NightActionResponse
// @Router /api/v1/game/night-action [post]
func (ctrl *WerewolfController) NightAction(c *gin.Context) {
	var req dto.NightActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	resp, err := ctrl.service.NightAction(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Vote 投票
// @Summary 投票
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.VoteRequest true "投票请求"
// @Success 200 {object} dto.VoteResponse
// @Router /api/v1/game/vote [post]
func (ctrl *WerewolfController) Vote(c *gin.Context) {
	var req dto.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	resp, err := ctrl.service.Vote(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetGameState 获取游戏状态
// @Summary 获取游戏状态
// @Tags Werewolf
// @Produce json
// @Param room_id query string true "房间ID"
// @Param player_id query string true "玩家ID"
// @Success 200 {object} dto.GameStateResponse
// @Router /api/v1/game/state [get]
func (ctrl *WerewolfController) GetGameState(c *gin.Context) {
	roomID := c.Query("room_id")
	playerID := c.Query("player_id")

	if roomID == "" || playerID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: "room_id and player_id are required",
		})
		return
	}

	resp, err := ctrl.service.GetGameState(c.Request.Context(), roomID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "service_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleWebSocket 处理 WebSocket 连接
// @Summary WebSocket 连接
// @Tags Werewolf
// @Param room_id query string true "房间ID"
// @Param player_id query string true "玩家ID"
// @Router /ws [get]
func (ctrl *WerewolfController) HandleWebSocket(c *gin.Context) {
	roomID := c.Query("room_id")
	playerID := c.Query("player_id")

	// 验证参数
	if roomID == "" || playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "room_id and player_id are required",
		})
		return
	}

	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// 注册客户端
	if err := ctrl.wsManager.RegisterClient(conn, roomID, playerID); err != nil {
		log.Printf("注册客户端失败: %v", err)
		conn.Close()
		return
	}

	log.Printf("WebSocket 连接建立: room=%s, player=%s", roomID, playerID)
}

// GetRoomPlayers 获取房间玩家列表
// @Summary 获取房间玩家列表
// @Tags Werewolf
// @Produce json
// @Param room_id query string true "房间ID"
// @Success 200 {object} dto.RoomPlayersResponse
// @Router /api/v1/rooms/players [get]
func (ctrl *WerewolfController) GetRoomPlayers(c *gin.Context) {
	roomID := c.Query("room_id")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: "room_id is required",
		})
		return
	}

	// 获取在线玩家数
	onlineCount := ctrl.wsManager.GetOnlinePlayerCount(roomID)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"room_id":      roomID,
		"online_count": onlineCount,
	})
}

// LeaveRoom 离开房间
// @Summary 离开房间
// @Tags Werewolf
// @Accept json
// @Produce json
// @Param request body dto.LeaveRoomRequest true "离开房间请求"
// @Success 200 {object} dto.Response
// @Router /api/v1/rooms/leave [post]
func (ctrl *WerewolfController) LeaveRoom(c *gin.Context) {
	var req dto.LeaveRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 断开 WebSocket 连接
	ctrl.wsManager.UnregisterClient(req.PlayerID)

	// 清除 Cookie
	c.SetCookie("player_id", "", -1, "/", "", false, true)
	c.SetCookie("room_id", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.Response{
		Success: true,
		Message: "已离开房间",
	})
}
