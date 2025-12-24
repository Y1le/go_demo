// internal/app/app.go
package app

import (
	"context"
	"io"
	"liam/internal/client"
	"liam/internal/controllers/market"
	"liam/internal/controllers/user"
	werewolf "liam/internal/controllers/werewolf"
	"liam/internal/websocket"
	"liam/pkg/middleware"
	"log"
	"os"
	"time"

	"liam/config"
	"liam/internal/cronjobs"
	"liam/internal/models"
	"liam/internal/routes"
	"liam/internal/services"
	"liam/repositories"
	"liam/utils"

	"github.com/gin-contrib/cors"
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
	// 启动定时任务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Connected to Redis!")

	// 7. Email
	emailSender := utils.NewEmailSender(&cfg.Email)

	// 8. DI
	userRepo := repositories.NewUserRepository(db)
	marketPriceRepo := repositories.NewMarketPriceRepository(db)
	redisRepo := repositories.NewRedisRepository(redisClient)
	emailService := services.NewEmailService(emailSender, redisRepo, userRepo)
	userService := services.NewUserService(userRepo, emailService)
	marketPriceService := services.NewMarketPriceService(marketPriceRepo)
	userController := user.NewUserController(userService, emailService)
	marketPriceController := market.NewMarketPriceController(marketPriceService)

	marketPriceService.CrawlAndSave()

	marketCron := cronjobs.NewMarketPriceCron(marketPriceService)
	if err := marketCron.Start(ctx); err != nil {
		log.Printf("Failed to start market price cron: %v", err)
	}

	grpcClient, err := client.NewWerewolfGRPCClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer grpcClient.Close()
	werewolfService := services.NewWerewolfService(grpcClient)
	wsHandler := websocket.NewWSHandler(grpcClient)
	wsManager := client.NewWSManager(grpcClient)
	werewolfController := werewolf.NewWerewolfController(werewolfService, wsManager)

	// 9. Router
	r := gin.Default()
	r.Use(middleware.RequestLogger()) // 确保 middleware 在 utils 或 internal/middleware

	// CORS 配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 注册路由（移到 routes 包）
	routes.PublicRoutes(r, userController)
	routes.ProtectedRoutes(r, userController)
	routes.MarketPriceRoutes(r, marketPriceController)
	routes.WolfGameRoutes(r, werewolfController, wsHandler)

	// 10. Start server
	return r.Run(":8080") // 或从 cfg 读取端口
}
