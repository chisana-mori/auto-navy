package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ServiceError 服务错误
type ServiceError struct {
	Code    int    // 错误码
	Message string // 错误信息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// 错误码定义
const (
	ErrCodeNotFound     = 404
	ErrCodeBadRequest   = 400
	ErrCodeServerError  = 500
	ErrCodeUnauthorized = 401
)

// NewNotFoundError 创建未找到错误
func NewNotFoundError(resource string, id int64) error {
	return &ServiceError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf(ErrRecordNotFoundMsg, resource, id),
	}
}

// NewBadRequestError 创建请求错误
func NewBadRequestError(message string) error {
	return &ServiceError{
		Code:    ErrCodeBadRequest,
		Message: message,
	}
}

// NewServerError 创建服务器错误
func NewServerError(message string, err error) error {
	return &ServiceError{
		Code:    ErrCodeServerError,
		Message: message,
		Err:     err,
	}
}

// IsNotFound 判断是否是未找到错误
func IsNotFound(err error) bool {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Code == ErrCodeNotFound
	}
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// HandleDBError 处理数据库错误
func HandleDBError(err error, resource string, id int64) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewNotFoundError(resource, id)
	}
	return NewServerError(fmt.Sprintf("database error when operating %s", resource), err)
} 