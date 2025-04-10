package service

// PaginationRequest 分页请求
type PaginationRequest struct {
	Page int `form:"page" json:"page" example:"1" swagger:"description=页码"`
	Size int `form:"size" json:"size" example:"10" swagger:"description=每页数量"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Page  int   `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int   `json:"size" example:"10" swagger:"description=每页数量"`
	Total int64 `json:"total" example:"100" swagger:"description=总记录数"`
}

// AdjustPagination 调整分页参数
func (p *PaginationRequest) AdjustPagination() {
	if p.Page <= 0 {
		p.Page = DefaultPage
	}
	if p.Size <= 0 || p.Size > MaxSize {
		p.Size = DefaultSize
	}
}

// GetOffset 获取偏移量
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.Size
}

// ToPaginationResponse 转换为分页响应
func (p *PaginationRequest) ToPaginationResponse(total int64) *PaginationResponse {
	return &PaginationResponse{
		Page:  p.Page,
		Size:  p.Size,
		Total: total,
	}
} 