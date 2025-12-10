// controllers/user/user_controller.go
package user

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
	userService  services.UserService
	emailService services.EmailService
}

// NewUserController 创建一个新的 UserController 实例
func NewUserController(userService services.UserService, emailService services.EmailService) *UserController {
	return &UserController{
		userService:  userService,
		emailService: emailService,
	}
}

// handleError 统一处理错误响应
func (uc *UserController) handleError(c *gin.Context, err error) {
	var appErr *errors.AppError
	if stdErr.As(err, &appErr) {
		statusCode := http.StatusInternalServerError
		switch appErr.Code {
		case errors.ErrNotFound.Code:
			statusCode = http.StatusNotFound
		case errors.ErrInvalidInput.Code:
			statusCode = http.StatusBadRequest
		case errors.ErrConflict.Code:
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Unwrap(),
		})
	} else {
		// 未知系统错误
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrInternalError.Code,
			"message": "An unexpected error occurred",
			"details": err.Error(),
		})
	}
}

func (uc *UserController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	// 4. 验证用户登录
	reap, err := uc.userService.Login(ctx, req.Email, req.Password)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}
	// 5. 返回登录成功响应
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Login successful.",
		"data":    reap,
	})
}

// RegisterUser 处理用户注册请求
func (uc *UserController) Register(ctx *gin.Context) {
	var req dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	userResp, err := uc.userService.RegisterUser(ctx, &req)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully, please check your email for verification code.",
		"user":    userResp,
	})
}

// SendVerificationEmail 处理发送验证码请求
func (uc *UserController) SendVerificationEmail(ctx *gin.Context) {
	var req dto.SendVerificationEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	err := uc.emailService.SendVerificationEmail(ctx, req.Email)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Verification email sent successfully."})
}

// VerifyEmail 处理邮箱验证请求
func (uc *UserController) VerifyEmail(ctx *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	err := uc.userService.VerifyUserEmail(ctx, req.Email, req.Code)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Email verified successfully."})
}

// CreateUser 处理创建用户的 HTTP 请求
func (uc *UserController) CreateUser(ctx *gin.Context) {
	var req dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	userResp, err := uc.userService.CreateUser(ctx.Request.Context(), &req)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, userResp)
}

// GetUserByID 根据 ID 获取用户
func (uc *UserController) GetUserByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	userResp, err := uc.userService.GetUserByID(ctx.Request.Context(), uint(id))
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, userResp)
}

// UpdateUser 更新用户信息
func (uc *UserController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	var req dto.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid request body", err))
		return
	}

	userResp, err := uc.userService.UpdateUser(ctx.Request.Context(), uint(id), &req)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, userResp)
}

// DeleteUser 删除用户（软删除）
func (uc *UserController) DeleteUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid user ID format", err))
		return
	}

	if err := uc.userService.DeleteUser(ctx.Request.Context(), uint(id)); err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User deleted successfully (soft delete)"})
}

// GetAllUsers 获取所有用户（支持分页）
func (uc *UserController) GetAllUsers(ctx *gin.Context) {
	var pagination dto.PaginationParams
	if err := ctx.ShouldBindQuery(&pagination); err != nil {
		uc.handleError(ctx, errors.NewAppError(errors.ErrInvalidInput.Code, "Invalid pagination parameters", err))
		return
	}

	usersResp, total, err := uc.userService.GetAllUser(ctx.Request.Context(), &pagination)
	if err != nil {
		uc.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":  usersResp,
		"total": total,
		"page":  pagination.Page,
		"limit": pagination.PageSize,
	})
}
