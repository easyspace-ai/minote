# Response 规范

本文档定义统一响应结构规范。

## 统一响应格式

所有接口必须返回此结构：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

## Response 结构定义

```go
package response

// Response 统一响应结构
type Response struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 状态码常量
const (
    CodeOK           = "OK"
    CodeBadRequest   = "BAD_REQUEST"
    CodeUnauthorized = "UNAUTHORIZED"
    CodeForbidden    = "FORBIDDEN"
    CodeNotFound     = "NOT_FOUND"
    CodeConflict     = "CONFLICT"
    CodeInternal     = "INTERNAL_ERROR"
)
```

## 响应函数

```go
package response

import (
    "encoding/json"
    "net/http"
)

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

// OK 成功响应
func OK(w http.ResponseWriter, data interface{}) {
    writeJSON(w, http.StatusOK, Response{
        Code:    CodeOK,
        Message: "success",
        Data:    data,
    })
}

// Created 创建成功响应
func Created(w http.ResponseWriter, data interface{}) {
    writeJSON(w, http.StatusCreated, Response{
        Code:    CodeOK,
        Message: "created",
        Data:    data,
    })
}

// NoContent 无内容响应
func NoContent(w http.ResponseWriter) {
    w.WriteHeader(http.StatusNoContent)
}

// BadRequest 请求错误响应
func BadRequest(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusBadRequest, Response{
        Code:    CodeBadRequest,
        Message: err.Error(),
    })
}

// Unauthorized 未授权响应
func Unauthorized(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusUnauthorized, Response{
        Code:    CodeUnauthorized,
        Message: err.Error(),
    })
}

// Forbidden 禁止访问响应
func Forbidden(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusForbidden, Response{
        Code:    CodeForbidden,
        Message: err.Error(),
    })
}

// NotFound 资源不存在响应
func NotFound(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusNotFound, Response{
        Code:    CodeNotFound,
        Message: err.Error(),
    })
}

// Conflict 冲突响应
func Conflict(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusConflict, Response{
        Code:    CodeConflict,
        Message: err.Error(),
    })
}

// InternalError 内部错误响应
func InternalError(w http.ResponseWriter, err error) {
    writeJSON(w, http.StatusInternalServerError, Response{
        Code:    CodeInternal,
        Message: "internal server error",  // 不暴露内部错误详情
    })
}
```

## 错误转换

```go
package response

import (
    "errors"
    "net/http"

    "github.com/easyspace-ai/minote/internal/service"
)

// FromError 根据错误类型返回对应响应
func FromError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, service.ErrUserNotFound):
        NotFound(w, err)
    case errors.Is(err, service.ErrUserAlreadyExists):
        Conflict(w, err)
    case errors.Is(err, service.ErrInvalidPassword):
        BadRequest(w, err)
    case errors.Is(err, service.ErrPermissionDenied):
        Forbidden(w, err)
    default:
        InternalError(w, err)
    }
}
```

## 分页响应

```go
// ListResponse 分页列表响应
type ListResponse struct {
    Items interface{} `json:"items"`
    Total int64       `json:"total"`
    Page  int         `json:"page"`
    Size  int         `json:"size"`
}

// OKList 列表成功响应
func OKList(w http.ResponseWriter, items interface{}, total int64, page, size int) {
    writeJSON(w, http.StatusOK, Response{
        Code:    CodeOK,
        Message: "success",
        Data: ListResponse{
            Items: items,
            Total: total,
            Page:  page,
            Size:  size,
        },
    })
}
```

## 在 Handler 中使用

```go
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, err)
        return
    }

    result, err := h.userService.Create(r.Context(), req)
    if err != nil {
        response.FromError(w, err)
        return
    }

    response.Created(w, result)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    var query ListUserQuery
    if err := parseQuery(r, &query); err != nil {
        response.BadRequest(w, err)
        return
    }

    result, err := h.userService.List(r.Context(), query)
    if err != nil {
        response.FromError(w, err)
        return
    }

    response.OKList(w, result.Items, result.Total, query.Page, query.PageSize)
}
```

## 禁止事项

```go
// ❌ 不统一的响应格式
json.NewEncoder(w).Encode(map[string]interface{}{"user": user})  // 禁止！

// ❌ 直接返回错误详情
w.WriteHeader(500)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})  // 禁止暴露内部错误！

// ❌ 不同接口返回不同结构
json.NewEncoder(w).Encode(user)  // 禁止！
json.NewEncoder(w).Encode(Response{Data: user})  // 另一个接口
```

## 相关文档

- [Handler 规范](./03-Handler规范.md)
- [错误处理规范](./05-错误处理规范.md)
