package repositories

import (
	"encoding/json"
	"fmt"
	"io"
	"liam/internal/dto/response"
	"liam/internal/models"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	baseURL        = "https://www.abuya.com.cn:6888/proPrice/ns/page.do"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	referer        = "https://www.abuya.com.cn:6888/webget/gzjg_new.html"
	xRequestedWith = "XMLHttpRequest"
	accept         = "application/json, text/javascript, */*; q=0.01"
)

type MarketPriceRepository interface {
	FetchTodayPrices(page, size int) (*response.DataType, error)
	CrawlAndSave() error
}

type MarketPriceRepositoryImpl struct {
	db             *gorm.DB
	httpClient     *http.Client
	baseURL        string
	userAgent      string
	referer        string
	xRequestedWith string
	accept         string
}

func NewMarketPriceRepository(db *gorm.DB) MarketPriceRepository {
	return &MarketPriceRepositoryImpl{
		db: db,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		baseURL:        baseURL,
		userAgent:      userAgent,
		referer:        referer,
		xRequestedWith: xRequestedWith,
		accept:         accept,
	}
}

func (r *MarketPriceRepositoryImpl) FetchTodayPrices(page, rows int) (*response.DataType, error) {
	// 构建 URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("rows", fmt.Sprintf("%d", rows))
	q.Set("params[state]", "3") // 模拟 params[state]=3
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Referer", r.referer)
	req.Header.Set("X-Requested-With", r.xRequestedWith)
	req.Header.Set("Accept", r.accept)
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd") // Go's http client handles this automatically

	// client := &http.Client{
	// 	Timeout: 10 * time.Second, // 设置超时
	// }

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d %s", resp.StatusCode, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// fmt.Printf("Raw JSON response (for debugging):\n%s\n", string(bodyBytes)) // 打印原始JSON以便调试

	var priceData response.DataType
	err = json.Unmarshal(bodyBytes, &priceData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &priceData, nil
}

func (r *MarketPriceRepositoryImpl) CrawlAndSave() error {
	currentPage := 1
	pageSize := 100
	maxPages := 5 // 假设我们只爬取前5页，实际可以根据 total 字段来决定
	for page := currentPage; page <= maxPages; page++ {
		fmt.Printf("Fetching page %d...\n", page)
		data, err := r.FetchTodayPrices(page, pageSize)
		if err != nil {
			log.Printf("Error fetching data for page %d: %v", page, err)
			// 根据错误类型决定是否继续爬取下一页
			continue
		}

		if data.Code != "0" {
			return fmt.Errorf("API returned error code %s: %s for page %d", data.Code, data.Msg, page)
		}

		if len(data.Rows) == 0 {
			return fmt.Errorf("No more data found.")
		}

		fmt.Printf("Total records: %d, Records on this page: %d\n", data.Total, len(data.Rows))
		var marketPrices []models.MarketPrice
		for _, row := range data.Rows {
			price, err := strconv.ParseFloat(row.Price, 64)
			if err != nil {
				log.Printf("Failed to parse price %s: %v", row.Price, err)
				continue
			}

			date, err := time.Parse("2006-01-02", row.PriceDate)
			if err != nil {
				log.Printf("Failed to parse price_date %s: %v", row.PriceDate, err)
				continue
			}

			marketPrices = append(marketPrices, models.MarketPrice{
				ProID:        row.ProID,
				ProName:      row.ProName,
				MarketID:     row.MarketID,
				MarketName:   row.MarketName,
				Price:        price,
				PriceUnit:    row.PriceUnit,
				SpecificiVal: row.SpecificiVal, // 如果接口返回了这个字段
				PriceDate:    date,
				CreateAt:     time.Now(),
			})
		}
		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(marketPrices, 100)
		if result.Error != nil {
			log.Printf("Error saving market prices: %v", result.Error)
		}
		log.Printf("Saved %d records", result.RowsAffected)
		// 如果总记录数小于当前页数 * 每页大小，说明已经爬取完所有数据
		if data.Total <= page*pageSize {
			return nil
		}

		time.Sleep(2 * time.Second) // 避免请求过快，给服务器减轻压力
	}

	return nil

}
