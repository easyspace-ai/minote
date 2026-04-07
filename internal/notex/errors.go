package notex

import (
	"encoding/json"
	"net/http"
)

// 错误码定义
const (
	ErrCodeSuccess           = 0
	ErrCodeBadRequest        = 400000
	ErrCodeUnauthorized      = 401000
	ErrCodeTokenExpired      = 401001
	ErrCodeForbidden         = 403000
	ErrCodeNotFound          = 404000
	ErrCodeConflict          = 409000
	ErrCodeInternalError     = 500000
	ErrCodeServiceUnavailable = 503000
)

// APIError 统一API错误类型
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error 实现error接口
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError 创建新的API错误
func NewAPIError(code int, message string, details ...string) *APIError {
	err := &APIError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// writeAPIResponse 统一API响应格式
func writeAPIResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]interface{}{
		"code":    status,
		"message": http.StatusText(status),
		"data":    data,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeAPIError 统一API错误响应
func writeAPIError(w http.ResponseWriter, status int, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]interface{}{
		"code":    err.Code,
		"message": err.Message,
		"error":   err,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
