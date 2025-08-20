package errors

import (
	"errors"
	"fmt"
)

type AppError struct {
	Code    int    //业务错误码
	Message string //错误消息
	Err     error  //原始错误
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code: %d, message: %s, original_error: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义的业务错误码和消息
var (
	ErrNotFound      = NewAppError(1001, "Resource not found", nil)
	ErrInvalidInput  = NewAppError(1002, "Invalid input parameters", nil)
	ErrInternalError = NewAppError(1003, "Internal server error", nil)
	ErrConflict      = NewAppError(1004, "Resource conflict", nil)
)

// IsNotFound 检查错误是否是 ErrNotFound
func IsNotFound(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == ErrNotFound.Code
}

// IsInvalidInput 检查错误是否是 ErrInvalidInput
func IsInvalidInput(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == ErrInvalidInput.Code
}

// func (m MyError) Is(target error) bool { return target == fs.ErrExist }
func Is(m error, target error) bool { return m == target }
