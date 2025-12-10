package repositories

import (
	"encoding/json"
	"fmt"
	"io"
	"liam/dto/response"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL   = "https://www.abuya.com.cn:6888/proPrice/ns/page.do"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	referer   = "https://www.abuya.com.cn:6888/webget/gzjg_new.html"
)

type MarketPriceRepository interface {
	FetchTodayPrices(page, size int) ([]*response.MarketPriceResp, error)
}

type MarketPriceRepositoryImpl struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	referer    string
}

func NewMarketPriceRepository() MarketPriceRepository {
	return &MarketPriceRepositoryImpl{
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		baseURL:   baseURL,
		userAgent: userAgent,
		referer:   referer,
	}
}

type dataType struct {
	Total int `json:"total"`
	Rows  []struct {
		ProID       string `json:"proID"`
		ProName     string `json:"proName"`
		MarketID    string `json:"marketID"`
		MarketName  string `json:"marketName"`
		Price       string `json:"tradePrice"`
		PriceUnit   string `json:"priceUnit"`
		SpecificVal string `json:"specificVal"`
		PriceDate   string `json:"priceDate"`
	} `json:"rows"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func (r *MarketPriceRepositoryImpl) FetchTodayPrices(page, size int) ([]*response.MarketPriceResp, error) {
	u, err := url.Parse(r.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("size", fmt.Sprintf("%d", size))
	q.Set("params[state]", "3")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Referer", r.referer)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data dataType
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if data.Code != "0" {
		return nil, fmt.Errorf("unexpected code: %s, msg: %s", data.Code, data.Msg)
	}

	var result []*response.MarketPriceResp
	for _, row := range data.Rows {
		price, err := strconv.ParseFloat(row.Price, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}
		priceDate, err := time.Parse("2006-01-02", row.PriceDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price date: %w", err)
		}

		result = append(result, &response.MarketPriceResp{
			ProName:     row.ProName,
			MarketName:  row.MarketName,
			SpecificVal: row.SpecificVal,
			Price:       price,
			PriceUnit:   row.PriceUnit,
			PriceDate:   priceDate,
		})
	}
	return result, nil
}
