// internal/app/app.go
package app

import (
	"context"
	"io"
	"liam/controllers/market"
	"liam/controllers/user"
	"liam/pkg/middleware"
	"log"
	"os"
	"time"

	"liam/config"
	"liam/internal/routes"
	"liam/models"
	"liam/repositories"
	"liam/services"
	"liam/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Run() error {

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// 3. Logger
	logWriter, err := utils.NewDailyLogWriter("logs", "runtime")
	if err != nil {
		log.Fatal(err)
	}
	defer logWriter.Close()

	multiWriter := io.MultiWriter(os.Stdout, logWriter)
	newLogger := logger.New(
		log.New(multiWriter, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// 4. DB
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{Logger: newLogger})
	if err != nil {
		return err
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 5. Migrate
	if err = db.AutoMigrate(&models.User{}); err != nil {
		return err
	}
	log.Println("Database migration completed!")

	// 6. Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err = redisClient.Ping(ctx).Result(); err != nil {
		return err
	}
	log.Println("Connected to Redis!")

	// 7. Email
	emailSender := utils.NewEmailSender(&cfg.Email)

	// 8. DI
	userRepo := repositories.NewUserRepository(db)
	marketPriceRepo := repositories.NewMarketPriceRepository()
	redisRepo := repositories.NewRedisRepository(redisClient)
	emailService := services.NewEmailService(emailSender, redisRepo, userRepo)
	userService := services.NewUserService(userRepo, emailService)
	marketPriceService := services.NewMarketPriceService(marketPriceRepo)
	userController := user.NewUserController(userService, emailService)
	marketPriceController := market.NewMarketPriceController(marketPriceService)

	// 9. Router
	r := gin.Default()
	r.Use(middleware.RequestLogger()) // 确保 middleware 在 utils 或 internal/middleware

	// 注册路由（移到 routes 包）
	routes.PublicRoutes(r, userController)
	routes.ProtectedRoutes(r, userController)
	routes.MarketPriceRoutes(r, marketPriceController)

	// 10. Start server
	return r.Run(":8080") // 或从 cfg 读取端口
}
