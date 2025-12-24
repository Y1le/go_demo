package routes

import (
	"liam/internal/controllers/market"
	"liam/internal/controllers/user"
	werewolf "liam/internal/controllers/werewolf"
	"liam/internal/websocket"
	"liam/utils"

	"github.com/gin-gonic/gin"
)

func PublicRoutes(r *gin.Engine, uc *user.UserController) {
	r.POST("/public/login", uc.Login)
	r.POST("/public/register", uc.Register)
}

func ProtectedRoutes(r *gin.Engine, uc *user.UserController) {
	auth := r.Group("/user")
	auth.Use(utils.AuthRequired())
}

func MarketPriceRoutes(r *gin.Engine, mc *market.MarketPriceController) {
	auth := r.Group("/market")
	auth.Use(utils.AuthRequired())
	auth.GET("/prices", mc.GetTodayPricese)
}

func WolfGameRoutes(r *gin.Engine, werewolfCtrl *werewolf.WerewolfController, wsHandler *websocket.WSHandler) {
	// API v1
	v1 := r.Group("/api/werewolf/v1")
	{
		// 房间路由
		rooms := v1.Group("/rooms")
		{
			rooms.POST("", werewolfCtrl.CreateRoom)
			rooms.POST("/join", werewolfCtrl.JoinRoom)
			rooms.POST("/start", werewolfCtrl.StartGame)
		}

		// 游戏路由
		game := v1.Group("/game")
		game.Use(utils.AuthRequired()) // 需要认证
		{
			game.POST("/night-action", werewolfCtrl.NightAction)
			game.POST("/vote", werewolfCtrl.Vote)
			game.GET("/state", werewolfCtrl.GetGameState)
			game.GET("/getrommplayers", werewolfCtrl.GetRoomPlayers)
			game.POST("/leave", werewolfCtrl.LeaveRoom)
		}
	}

	// WebSocket
	r.GET("/ws", wsHandler.HandleWebSocket)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
