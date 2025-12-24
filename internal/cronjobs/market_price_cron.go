package cronjobs

import (
	"context"
	"liam/internal/services"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type MarketPriceCron struct {
	service services.MarketPriceService
	cron    *cron.Cron
}

func NewMarketPriceCron(service services.MarketPriceService) *MarketPriceCron {
	return &MarketPriceCron{
		service: service,
		cron:    cron.New(cron.WithLocation(time.Local)),
	}
}

func (c *MarketPriceCron) Start(ctx context.Context) error {
	_, err := c.cron.AddFunc("0 2 * * *", func() {
		log.Printf("[Cron] starting market price crawl...")
		if err := c.service.CrawlAndSave(); err != nil {
			log.Printf("[Cron] Faild to crawl market price: %v", err)
		}
		log.Printf("[Cron] Market price crawl completed successfully")
	})
	if err != nil {
		return err
	}

	c.cron.Start()
	log.Printf("[Cron] Market price cron job started")

	go func() {
		<-ctx.Done()
		log.Printf("[Cron] Shutting done market price scheduler")
		c.cron.Stop()
	}()

	return nil
}
