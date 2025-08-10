package utils

import (
"log"
"net/http"
"time"

"github.com/gin-gonic/gin"
"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("hlfsdajfjaofdklafj;ajoiovnnv")

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error"	:	"Authorization header required",
			})
			return 
		}

		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error"	:	"Invalid Authorization header format. Expected 'Bearer <token>'",
			})
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok{
				return nil, jwt.ErrSignatureInvalid 
			}
			return jwtSecret, nil
		})

		if err != nil {
			log.Printf("JWT parsing error: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error" :	"Invalid or expired token",
			})
			return 
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error" :	"Invalid token",
			})
			return 
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {

			log.Printf("JWT Claims: %+v", claims) // 打印所有解析出的 claims
            
            // 检查 user_id 是否存在并且非空
            if _, exists := claims["user_id"]; !exists {
                log.Println("Warning: 'user_id' claim not found in JWT token.")
            } else {
                log.Printf("Setting user_id in context: %v", claims["user_id"])
            }
			c.Set("user_id", claims["user_id"])
			c.Set("user_name",claims["user_name"])
		}else {
            log.Println("Error: Token claims could not be cast to jwt.MapClaims.")
        }
		// 令牌有效，继续处理请求
		c.Next()
	}
}

func GenerateToken(userID string, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id" :	userID,
		"user_name"	: username,
		"exp" : time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}