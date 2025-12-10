package services

import (
	"liam/dto/response"
	"liam/repositories"
)

type MarketPriceService interface {
	GetMarketPrices(page, size int) ([]*response.MarketPriceResp, int, error)
}

type MarketPriceServiceImpl struct {
	repo repositories.MarketPriceRepository
}

func NewMarketPriceService(repo repositories.MarketPriceRepository) *MarketPriceServiceImpl {
	return &MarketPriceServiceImpl{repo: repo}
}

func (s *MarketPriceServiceImpl) GetMarketPrices(page, size int) ([]*response.MarketPriceResp, int, error) {
	datas, err := s.repo.FetchTodayPrices(page, size)
	if err != nil {
		return nil, 0, err
	}
	return datas, len(datas), nil
}
