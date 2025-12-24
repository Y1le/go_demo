package models

import (
	"time"
)

type MarketPrice struct {
	ID           uint      `json:"id,omitempty gorm:"primaryKey"`
	ProID        string    `json:"pro_id"`
	ProName      string    `json:"pro_name"`
	MarketID     string    `json:"market_id"`
	MarketName   string    `json:"market_name"`
	Price        float64   `json:"price"`
	PriceUnit    string    `json:"price_unit"`
	SpecificiVal string    `json:"specifici_val"`
	PriceDate    time.Time `json:"price_date"`
	CreateAt     time.Time `json:"create_at,omitempty gorm:"autoCreateTime"`
}
