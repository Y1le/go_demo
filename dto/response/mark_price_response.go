package response

import (
	"time"

	"liam/models"
)

type MarketPriceResp struct {
	ProName     string    `json:"pro_name"`
	MarketName  string    `json:"market_name"`
	Price       float64   `json:"price"`
	PriceUnit   string    `json:"price_unit"`
	SpecificVal string    `json:"specifici_val"`
	PriceDate   time.Time `json:"price_date"`
}

func FromModel(m *models.MarketPrice) *MarketPriceResp {
	return &MarketPriceResp{
		ProName:     m.ProName,
		MarketName:  m.MarketName,
		Price:       m.Price,
		PriceUnit:   m.PriceUnit,
		SpecificVal: m.SpecificVal,
		PriceDate:   m.PriceDate,
	}
}
