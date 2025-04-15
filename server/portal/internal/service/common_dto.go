package service

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
