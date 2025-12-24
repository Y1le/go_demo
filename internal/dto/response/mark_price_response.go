package response

import (
	"liam/internal/models"
	"time"
)

type MarketPriceResp struct {
	ProID        string    `json:"proID"`
	ProName      string    `json:"proName"` // 品种名称
	ProTypeID    string    `json:"proTypeID"`
	MarketID     string    `json:"marketID"`
	MarketName   string    `json:"marketName"`   // 市场名称
	Price        float64   `json:"tradePrice"`   // 价格 (假设是 tradePrice)
	PriceUnit    string    `json:"priceUnit"`    // 价格单位
	PriceDate    time.Time `json:"priceDate"`    // 报价日期
	SpecificiVal string    `json:"specificiVal"` // 规格
}

type DataType struct {
	Total int `json:"total"`
	Rows  []struct {
		ProID        string `json:"proID"`
		ProName      string `json:"proName"`
		MarketID     string `json:"marketID"`
		MarketName   string `json:"marketName"`
		Price        string `json:"tradePrice"`
		PriceUnit    string `json:"priceUnit"`
		SpecificiVal string `json:"specificiVal"`
		PriceDate    string `json:"priceDate"`
	} `json:"rows"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func FromModel(m *models.MarketPrice) *MarketPriceResp {
	return &MarketPriceResp{
		ProID:        m.ProID,
		ProName:      m.ProName,
		MarketID:     m.MarketID,
		MarketName:   m.MarketName,
		Price:        m.Price,
		PriceUnit:    m.PriceUnit,
		SpecificiVal: m.SpecificiVal,
		PriceDate:    m.PriceDate,
	}
}
