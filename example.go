package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"liam/config"
	"liam/controllers/upload"
	controllers "liam/controllers/user"
	"liam/models"
	"liam/services"
	"liam/utils"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	//Initialize the connection DB
	config.InitDB()

	err = config.DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to auto migrate:%v ", err)
	}
	fmt.Println("Database migration successful!")

	userService := services.NewUserService(config.DB)

	userController := controllers.NewUserController(userService)

	r := gin.Default()
	// 注册全局中间件
	// r.Use(middleware.RequestLogger()) // 自定义请求日志中间件

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
