package dto

type CreateUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Age      int    `json:"age" binding:"gte=0,lte=150"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"` // 使用指针表示可选字段
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	Age      *int    `json:"age,omitempty" binding:"omitempty,gte=0,lte=150"`
	Password string  `json:"password,omitempty" binding:"omitempty,required,min=6"`
}

type UserResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PaginationParams struct {
	Page     int `form:"page,default=1" binding:"gte=1"`
	PageSize int `form:"page_size,default=10" banding:"gte=1,lte=100"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Name            string `json:"name" binding:"required,min=2,max=100"`
	Email           string `json:"email" binding:"required,email"`
	Age             int    `json:"age" binding:"gte=0,lte=150"`
	Password        string `json:"password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

// SendVerificationEmailRequest 用于发送验证码请求
type SendVerificationEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyEmailRequest 用于邮箱验证码验证请求
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}
