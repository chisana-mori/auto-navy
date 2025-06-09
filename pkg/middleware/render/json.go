package render

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 通用响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// ErrorResponse 错误响应
// swagger:model
type ErrorResponse struct {
	Error string `json:"error" example:"操作失败" swagger:"description=错误消息"`
}

// SuccessResponse 成功响应
// swagger:model
type SuccessResponse struct {
	Message string `json:"message" example:"操作成功" swagger:"description=成功消息"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: http.StatusOK,
		Msg:  "success",
		Data: data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: http.StatusOK,
		Msg:  message,
		Data: data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, Response{
		Code: httpCode,
		Msg:  message,
	})
}

// FailWithData 带数据的失败响应
func FailWithData(c *gin.Context, httpCode int, message string, data interface{}) {
	c.JSON(httpCode, Response{
		Code: httpCode,
		Msg:  message,
		Data: data,
	})
}

// BadRequest 400错误响应
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, message)
}

// Unauthorized 401错误响应
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, message)
}

// Forbidden 403错误响应
func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, message)
}

// NotFound 404错误响应
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, message)
}

// InternalServerError 500错误响应
func InternalServerError(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, message)
}
