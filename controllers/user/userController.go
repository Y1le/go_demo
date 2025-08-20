package controllers

import (
	"liam/services"
	"net/http"

	"liam/errors"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService services.UserService
}

func NewUserController(userService services.UserService) *UserController {
	return &UserController{userService: userService}
}

// handleError 统一处理错误响应
func (ctrl *UserController) handleError(c *gin.Context, err error) {
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case errors.ErrNotFound.Code:
			c.JSON(http.StatusNotFound, gin.H{"code": appErr.Code, "message": appErr.Message})
		case errors.ErrInvalidInput.Code:
			c.JSON(http.StatusBadRequest, gin.H{"code": appErr.Code, "message": appErr.Message, "details": appErr.Unwrap().Error()})
		case errors.ErrConflict.Code:
			c.JSON(http.StatusConflict, gin.H{"code": appErr.Code, "message": appErr.Message})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": appErr.Code, "message": appErr.Message, "details": appErr.Unwrap().Error()})
		}
	} else {
		// 未知错误
		c.JSON(http.StatusInternalServerError, gin.H{"code": errors.ErrInternalError.Code, "message": "An unexpected error occurred", "details": err.Error()})
	}
}

// func (ctrl *UserController) CreateUser(c *gin.Context) {
// 	var user models.User
// 	if err := c.ShouldBindJSON(&user); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := ctrl.userService.CreateUser(&user); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failedd to created user"})
// 		return
// 	}
// 	c.JSON(http.StatusCreated, user)
// }

// func (ctrl *UserController) GetAllUser(c *gin.Context) {
// 	users, err := ctrl.userService.GetAllUser()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, users)
// }

// func (ctrl *UserController) GetUserByID(c *gin.Context) {
// 	idStr := c.Param("id")
// 	id, err := strconv.ParseUint(idStr, 10, 32)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
// 	}

// 	user, err := ctrl.userService.GetUserByID(uint(id))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error": "Failed to retrieve user",
// 		})
// 	}
// 	if user == nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"error": "User not found",
// 		})
// 	}
// 	c.JSON(http.StatusOK, user)
// }

// func (ctrl *UserController) UpdateUser(c *gin.Context) {
// 	idStr := c.Param("id")
// 	id, err := strconv.ParseUint(idStr, 10, 32)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "Invaild user ID",
// 		})
// 		return
// 	}

// 	user, err := ctrl.userService.GetUserByID(uint(id))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error": "Failed to retrieve user for update",
// 		})
// 		return
// 	}
// 	if user == nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"error": "User not found",
// 		})
// 		return
// 	}

// 	if err := c.ShouldBindJSON(&user); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	if err := ctrl.userService.UpdateUser(user); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error": "Failed to uptate user",
// 		})
// 		return
// 	}
// 	c.JSON(http.StatusOK, user)

// }

// func (ctrl *UserController) DeleteUser(c *gin.Context) {
// 	idStr := c.Param("id")

// 	id, err := strconv.ParseUint(idStr, 10, 32)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
// 		return
// 	}

// 	err = ctrl.userService.DeleteUser(uint(id))
// 	if err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "User not found or already deleted"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
// 		}
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "User delete successfully",
// 	})
// }
