package routes

import (
	"liam/controllers/market"
	"liam/controllers/user"
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
