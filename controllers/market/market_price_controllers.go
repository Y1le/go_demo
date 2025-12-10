package market

import (
	"liam/dto/request"
	"liam/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MarketPriceController struct {
	service services.MarketPriceService
}

func NewMarketPriceController(service services.MarketPriceService) *MarketPriceController {
	return &MarketPriceController{service: service}
}

func (ctl *MarketPriceController) GetTodayPricese(c *gin.Context) {
	var req request.GetMarketPriceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	prices, total, err := ctl.service.GetMarketPrices(req.Page, req.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取市场价格失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  prices,
		"total": total,
		"page":  req.Page,
		"size":  req.Size,
	})
}
