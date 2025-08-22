package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"liam/config"
	"liam/controllers/upload"
	controllers "liam/controllers/user"
	"liam/models"
	"liam/pkg/middleware"
	"liam/repositories"
	"liam/services"
	"liam/utils"
)

func main() {
	// 1. 加载配置
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接可复用的最长时间
	// 自动迁移模型
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}
	fmt.Println("Database migration completed!")
	// Initialize the connection DB
	// db, err := config.InitDB()
	// 自动迁移数据库表
	// err = config.AutoMigrate(db, &models.User{}) // 传入模型
	// if err != nil {
	// 	log.Fatalf("Failed to auto migrate database: %v", err)
	// }
	// fmt.Println("Database migration successful!")

	// 3. 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis!")

	// 4. 初始化邮件发送器
	emailSender := utils.NewEmailSender(&cfg.Email)

	r := gin.Default()
	// 注册全局中间件
	r.Use(middleware.RequestLogger()) // 自定义请求日志中间件
	// 依赖注入
	userRepo := repositories.NewUserRepository(db)
	redisRepo := repositories.NewRedisRepository(redisClient)
	emailService := services.NewEmailService(emailSender, redisRepo, userRepo)
	userService := services.NewUserService(userRepo, emailService)
	userController := controllers.NewUserController(userService, emailService)

	r.POST("/public/login", loginEndpoint)

	authorized := r.Group("/")
	authorized.Use(utils.AuthRequired())
	{
		authorized.POST("/submit", submitEndpoint)
		authorized.POST("/read", readEndpoint)
		// v1 := authorized.Group("/v1")
		// v2 := authorized.Group("/v2")
		testing := authorized.Group("testing")
		testing.GET("/analytics", analyticsEndpoint)

	}

	//上传网站文件
	r.GET("/upload_url", upload.Upload)

	r.GET("/local/file", func(c *gin.Context) {
		c.File("example.go")
	})

	r.POST("/post", func(c *gin.Context) {
		id := c.Query("id")
		name := c.Query("name")

		formName := c.PostForm("name")
		formMessage := c.PostForm("message")

		c.JSON(200, gin.H{
			"status":      "success",
			"id":          id,
			"name":        name,
			"formName":    formName,
			"formMessage": formMessage,
		})
	})

	r.GET("/user/:name", func(c *gin.Context) {
		random := rand.Intn(1000)
		name := c.Param("name")
		res := fmt.Sprintf("Hello %s your lucynumber is %d!", name, random)
		c.JSON(http.StatusOK, gin.H{
			"hah": res,
		})
	})

	r.GET("/user/:name/*action", func(c *gin.Context) {
		name := c.Param("name")
		action := c.Param("action")
		message := name + " is " + action
		c.String(http.StatusOK, message)
	})

	r.GET("/getContext", func(c *gin.Context) {
		// clientIP := c.ClientIP() // 获取客户端 IP
		clientIP := "240a:42c6:9c02:b4c:1098:44ff:feec:90de"
		// 调用 utils 包中的 GetGeolocation 函数
		ipInfo, err := utils.GetGeolocation(clientIP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get geolocation: %v", err)})
			return
		}

		response := gin.H{
			"request_method":  c.Request.Method,
			"request_path":    c.Request.URL.Path,
			"query_params":    c.Request.URL.RawQuery,
			"client_ip":       clientIP,
			"user_agent":      c.Request.UserAgent(),
			"full_path_route": c.FullPath(),
			"headers":         c.Request.Header,
			"geolocation":     ipInfo, // 将地理位置信息添加到响应中
		}

		c.JSON(http.StatusOK, response)
	})

	r.POST("/sendEmail", func(c *gin.Context) {
		var json map[string]interface{}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		email, ok := json["email"].(string)
		if !ok {
			c.JSON(400, gin.H{"error": "Email field is missing or not a string"})
			return
		}
		// if err := emailService.SendVerificationEmail(ctx, email); err != nil {
		// 	c.JSON(400, gin.H{"error": err})
		// 	return
		// }
		if err := emailSender.SendEmail(email, "title", "123456"); err != nil {
			c.JSON(400, gin.H{"error": err})
			return
		}
		c.JSON(200, gin.H{"message": "Email received successfully", "email": email})
	})

	r.POST("/user/:name/*action", func(c *gin.Context) {
		name := c.Param("name")
		action := c.Param("action") // 获取通配符参数匹配到的实际值

		// 打印注册路由时的完整路径模式
		fullPath := c.FullPath()

		c.String(http.StatusOK, "Full Path: %s\nUser: %s\nAction: %s", fullPath, name, action)
	})

	{
		v1 := r.Group("/v1")
		v1.POST("/login", loginEndpoint)
		v1.POST("/submit", submitEndpoint)
		v1.POST("/read", readEndpoint)
	}

	// Simple group: v2
	{
		v2 := r.Group("/v2")
		v2.POST("/login", loginEndpoint)
		v2.POST("/submit", submitEndpoint)
		v2.POST("/read", readEndpoint)
	}

	userRoutes := r.Group("/users")
	{
		// 注册和邮箱验证相关路由
		userRoutes.POST("/register", userController.RegisterUser)
		userRoutes.POST("/send-verification-email", userController.SendVerificationEmail)
		userRoutes.POST("/verify-email", userController.VerifyEmail)

		userRoutes.POST("/", userController.CreateUser)
		userRoutes.GET("/", userController.GetAllUser)
		userRoutes.GET("/:id", userController.GetUserByID)
		userRoutes.PUT("/:id", userController.UpdateUser)
		userRoutes.DELETE("/:id", userController.DeleteUser)
	}

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

type LoginRequest struct {
	ID       string `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
}

func loginEndpoint(c *gin.Context) {
	var req LoginRequest // 声明结构体变量

	// 绑定 JSON 请求体到结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := utils.GenerateToken(req.ID, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "token": token})
}

// submitEndpoint 模拟一个需要授权的 POST 请求
func submitEndpoint(c *gin.Context) {
	// 可以从 Context 中获取 AuthRequired 中间件设置的用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("user_name")
	c.JSON(http.StatusOK, gin.H{"message": "Data submitted successfully", "user_id": userID, "username": username})
}

func readEndpoint(c *gin.Context) {
	userID, _ := c.Get("user_id")
	c.JSON(http.StatusOK, gin.H{"message": "Data read successfully", "user_id": userID})
}

func analyticsEndpoint(c *gin.Context) {
	usernameAny, exists := c.Get("user_name")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User information not found in context"})
		return
	}

	username, ok := usernameAny.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user_name type in context"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Analytics data for " + username})
}
