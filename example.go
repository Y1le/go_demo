package main

import (
  "net/http"
  "fmt"
  "math/rand"
  "github.com/gin-gonic/gin"
  "log"
  "liam/utils"
  "os"
  "mime/multipart"
  "github.com/aliyun/aliyun-oss-go-sdk/oss"
  "github.com/joho/godotenv"
)

func saveToAliyunOSS(file *multipart.FileHeader, bucketName, objectName string) error {
    // 创建 OSS 客户端
    accessKeyId := os.Getenv("OSS_ACCESS_KEY_ID")
    accessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
    fmt.Printf("accessKeyId : %s\n",accessKeyId)
    fmt.Printf("accessKeySecret : %s\n",accessKeySecret)
  

    client, err := oss.New("oss-cn-heyuan.aliyuncs.com", accessKeyId, accessKeySecret)
    if err != nil {
        return err
    }

    // 获取存储空间
    bucket, err := client.Bucket(bucketName)
    if err != nil {
        return err
    }

    // 打开上传的文件
    srcFile, err := file.Open()
    if err != nil {
        return err
    }
    defer srcFile.Close()

    // 上传到 OSS
    err = bucket.PutObject(objectName, srcFile)
    if err != nil {
        return err
    }

    log.Printf("File %s uploaded to OSS bucket %s\n", objectName, bucketName)
    return nil
}

func main() {
  err := godotenv.Load()
  if err != nil {
      log.Fatalf("Error loading .env file: %v", err)
  }
  r := gin.Default()

  r.POST("/public/login", loginEndpoint)

  authorized := r.Group("/")
  authorized.Use(utils.AuthRequired())
  {
    authorized.POST("/submit", submitEndpoint)
    authorized.POST("/read", readEndpoint)

    testing := authorized.Group("testing")
    testing.GET("/analytics", analyticsEndpoint)

  }

  r.POST("/form_post", func(c *gin.Context){
    form, err := c.MultipartForm()
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid form data"})
        return
    }
    // 获取普通字段
    username := form.Value["username"][0]  // 假设字段名是 "username"
    password := form.Value["password"][0]        // 假设字段名是 "password"

    // 处理文件（如果有）
    files := form.File["files"]  // 假设文件字段名是 "files"
    for _, file := range files {
        log.Println("Received file:", file.Filename)
        c.SaveUploadedFile(file, "./uploads/"+file.Filename)
    }
    for _, file := range files {
      log.Println("Uploading file to Aliyun OSS:", file.Filename)
      err := saveToAliyunOSS(file, "liam-bucket", "uploads/"+file.Filename)
      if err != nil {
          c.JSON(500, gin.H{"error": "OSS初始化失败: " + err.Error()})
          return
      }
  }
    c.JSON(200, gin.H{
        "status":  "posted",
        "username": username,
        "password":    password,
        "files":   len(files),
    })
  })
  r.POST("/post", func(c *gin.Context){
    id := c.Query("id")
    name := c.Query("name")

    formName := c.PostForm("name")
    formMessage := c.PostForm("message")

    c.JSON(200, gin.H{
      "status"  : "success",
      "id"      : id,
      "name"    : name,
      "formName": formName,
      "formMessage" : formMessage,
    })
  })
  r.GET("/ping", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
      "message": "pong",
      "haha" : hah(),
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


  r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func hah() string{
  return "haha"
}
type LoginRequest struct {
    ID       string    `json:"id" binding:"required"`
    Username string `json:"username" binding:"required"`
}

func loginEndpoint(c *gin.Context) {
    var req LoginRequest // 声明结构体变量

    // 绑定 JSON 请求体到结构体
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // 绑定失败，返回错误
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
    username, _ := c.Get("username")
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