package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ApiResponse 统一API响应结构
type ApiResponse struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, ApiResponse{
		Code:    httpCode,
		Message: message,
		Data:    nil,
	})
}

// 分页响应结构
type PaginationResponse struct {
	List       interface{} `json:"list"`        // 数据列表
	Total      int64       `json:"total"`       // 总数
	Page       int         `json:"page"`        // 当前页
	PageSize   int         `json:"page_size"`   // 每页大小
	TotalPages int         `json:"total_pages"` // 总页数
}

// PaginationSuccessResponse 分页成功响应
func PaginationSuccessResponse(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	totalPages := int(total)/pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "success",
		Data: PaginationResponse{
			List:       list,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// 常见错误响应函数
func BadRequestError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

func UnauthorizedError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

func ForbiddenError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, message)
}

func NotFoundError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

func ConflictError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, message)
}

func InternalServerError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}