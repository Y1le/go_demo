package errors

import (
	stdErrors "errors" // <--- 导入 Go 标准库的 errors 包，并使用别名 stdErrors
	"fmt"
)

// AppError 是应用程序的自定义错误类型
// 它封装了业务错误码、用户消息和可选的底层原始错误。
type AppError struct {
	Code    int    // 业务错误码，用于区分不同的业务错误类型
	Message string // 用户友好的错误消息，可以直接返回给客户端
	Err     error  // 原始底层错误（可选），用于内部日志记录和调试
}

// Error 方法实现了 Go 的 error 接口。
// 当错误被打印或转换为字符串时，会调用此方法。
func (e *AppError) Error() string {
	if e.Err != nil {
		// 如果有底层错误，打印更详细的信息
		return fmt.Sprintf("code: %d, message: %s, original_error: %v", e.Code, e.Message, e.Err)
	}
	// 如果没有底层错误，只打印业务码和消息
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// Unwrap 方法实现了 Go 1.13+ 的错误链接口。
// 它返回 AppError 封装的原始底层错误。
// 这使得标准库的 errors.Is() 和 errors.As() 函数能够遍历错误链。
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 是一个工厂函数，用于创建 AppError 实例。
// code: 业务错误码
// message: 用户友好的错误消息
// err: 原始底层错误（可以是 nil）
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// --- 预定义的业务错误 ---
// 这些是常用的 AppError 实例，可以直接在代码中使用，提高一致性。

var (
	// ErrNotFound 表示资源未找到错误
	ErrNotFound = NewAppError(1001, "Resource not found", nil)

	// ErrInvalidInput 表示请求参数无效错误
	ErrInvalidInput = NewAppError(1002, "Invalid input parameters", nil)

	// ErrInternalError 表示服务器内部错误
	ErrInternalError = NewAppError(1003, "Internal server error", nil)

	// ErrConflict 表示资源冲突错误（例如唯一约束冲突）
	ErrConflict = NewAppError(1004, "Resource conflict", nil)

	// ErrUnauthorized 表示未授权错误
	ErrUnauthorized = NewAppError(1005, "Unauthorized", nil)

	// ErrForbidden 表示禁止访问错误
	ErrForbidden = NewAppError(1006, "Forbidden", nil)
)

// --- 辅助函数，用于检查错误类型 ---
// 这些函数封装了标准库的 errors.As()，使得检查 AppError 更加方便。

// IsNotFound 检查给定的错误是否是 ErrNotFound 类型。
func IsNotFound(err error) bool {
	var appErr *AppError
	// stdErrors.As 尝试将 err 链中的任何错误转换为 *AppError 类型
	// 如果成功，并且其 Code 与 ErrNotFound.Code 匹配，则返回 true
	return stdErrors.As(err, &appErr) && appErr.Code == ErrNotFound.Code
}

// IsInvalidInput 检查给定的错误是否是 ErrInvalidInput 类型。
func IsInvalidInput(err error) bool {
	var appErr *AppError
	return stdErrors.As(err, &appErr) && appErr.Code == ErrInvalidInput.Code
}

// IsInternalError 检查给定的错误是否是 ErrInternalError 类型。
func IsInternalError(err error) bool {
	var appErr *AppError
	return stdErrors.As(err, &appErr) && appErr.Code == ErrInternalError.Code
}

func IsCustomError(err error, target *AppError) bool {
	var appErr *AppError
	// 尝试将 err 链中的任何错误转换为 *AppError 类型
	// 如果成功转换，并且其 Code 与 target 的 Code 匹配，则返回 true
	return stdErrors.As(err, &appErr) && appErr.Code == target.Code
}

// IsConflict 检查给定的错误是否是 ErrConflict 类型。
func IsConflict(err error) bool {
	var appErr *AppError
	return stdErrors.As(err, &appErr) && appErr.Code == ErrConflict.Code
}

// IsUnauthorized 检查给定的错误是否是 ErrUnauthorized 类型。
func IsUnauthorized(err error) bool {
	var appErr *AppError
	return stdErrors.As(err, &appErr) && appErr.Code == ErrUnauthorized.Code
}

// IsForbidden 检查给定的错误是否是 ErrForbidden 类型。
func IsForbidden(err error) bool {
	var appErr *AppError
	return stdErrors.As(err, &appErr) && appErr.Code == ErrForbidden.Code
}

// AsAppError 尝试将给定的错误转换为 *AppError 类型。
// 如果成功转换，它将填充 target 并返回 true；否则返回 false。
// 这是对标准库 errors.As 的一个通用封装，用于你的 AppError 类型。
func AsAppError(err error, target **AppError) bool {
	return stdErrors.As(err, target)
}
