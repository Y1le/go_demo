package controllers

import (
	"liam/dto"
	"liam/pkg/errors"
	"liam/services"
	"net/http"
	"strconv"

	stdErr "errors"

	"github.com/gin-gonic/gin"
)

// UserController 定义用户控制器
type UserController struct {
	userService services.UserService
}

// NewUserController 创建一个新的 UserController 实例
func NewUserController(userService services.UserService) *UserController {
	return &UserController{userService: userService}
}

// handleError 统一处理错误响应
func (ctrl *UserController) handleError(c *gin.Context, err error) {
	var appErr *errors.AppError
	if stdErr.As(err, &appErr) {
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

// CreateUser 处理创建用户的 HTTP 请求
func (ctrl *UserController) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	userResp, err := ctrl.userService.CreateUser(c.Request.Context(), &req) // 传递 context
	if err != nil {
		ctrl.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, userResp)
}

// GetUserByID 处理根据 ID 获取用户的 HTTP 请求
func (ctrl *UserController) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	userResp, err := ctrl.userService.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		ctrl.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, userResp)
}

// UpdateUser 处理更新用户的 HTTP 请求
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	userResp, err := ctrl.userService.UpdateUser(c.Request.Context(), uint(id), &req)
	if err != nil {
		ctrl.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, userResp)
}

// DeleteUser 处理删除用户的 HTTP 请求
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	if err := ctrl.userService.DeleteUser(c.Request.Context(), uint(id)); err != nil {
		ctrl.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully (soft delete)"})
}

// GetAllUsers 处理获取所有用户的 HTTP 请求，支持分页
func (ctrl *UserController) GetAllUser(c *gin.Context) {
	var pagination dto.PaginationParams
	// 使用 ShouldBindQuery 来绑定查询参数
	if err := c.ShouldBindQuery(&pagination); err != nil {
		ctrl.handleError(c, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid pagination parameters", err))
		return
	}

	usersResp, total, err := ctrl.userService.GetAllUser(c.Request.Context(), &pagination)
	if err != nil {
		ctrl.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  usersResp,
		"total": total,
		"page":  pagination.Page,
		"limit": pagination.PageSize,
	})
}
