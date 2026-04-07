# Handler 规范

本文档定义标准库 net/http Handler 的编写规范。

## Handler 职责

Handler **只能做 5 件事**：

1. 参数绑定（Bind）
2. 参数校验（Validate）
3. 调用 Service
4. 错误转换
5. 返回统一 Response

## 标准模板

```go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/easyspace-ai/minote/internal/notex"
    "github.com/easyspace-ai/minote/pkg/response"
)

type UserHandler struct {
    userService notex.UserService
}

func NewUserHandler(userService notex.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// Create 创建用户
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. 参数绑定
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, err)
        return
    }

    // 2. 参数校验（如果 json tag 不够用）
    if err := req.Validate(); err != nil {
        response.BadRequest(w, err)
        return
    }

    // 3. 调用 Service
    result, err := h.userService.Create(r.Context(), req)
    
    // 4. 错误转换
    if err != nil {
        response.FromError(w, err)
        return
    }

    // 5. 返回统一 Response
    response.OK(w, result)
}
```

## 参数绑定

### JSON Body

```go
var req CreateUserRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    response.BadRequest(w, err)
    return
}
```

### URL 参数 (Go 1.22+)

```go
id := r.PathValue("id")
if id == "" {
    response.BadRequest(w, errors.New("id is required"))
    return
}
```

### Query 参数

```go
query := ListUserQuery{
    Page:     parseInt(r.URL.Query().Get("page"), 1),
    PageSize: parseInt(r.URL.Query().Get("page_size"), 20),
}
if err := query.Validate(); err != nil {
    response.BadRequest(w, err)
    return
}
```

## Request 结构定义

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Password string `json:"password" validate:"required,min=8"`
}

// Validate 自定义校验（binding tag 不够用时）
func (r *CreateUserRequest) Validate() error {
    // 复杂校验逻辑
    return nil
}
```

## Context 传递

始终使用 `r.Context()` 传递 context：

```go
result, err := h.userService.Create(r.Context(), req)
```

## 路由注册 (Go 1.22+ 模式)

```go
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("POST /users", h.Create)
    mux.HandleFunc("GET /users", h.List)
    mux.HandleFunc("GET /users/{id}", h.Get)
    mux.HandleFunc("PUT /users/{id}", h.Update)
    mux.HandleFunc("DELETE /users/{id}", h.Delete)
}
```

## 禁止事项

```go
// ❌ 在 handler 中写业务判断
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    if user.Age < 18 {  // 业务逻辑应在 Service
        response.BadRequest(w, errors.New("age must be >= 18"))
        return
    }
}

// ❌ 在 handler 中操作数据库
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    h.db.Create(&user)  // 禁止！
}

// ❌ 在 handler 中开启事务
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    tx := h.db.Begin()  // 禁止！
}

// ❌ 直接返回 Model
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    response.OK(w, userModel)  // 应返回 DTO
}
```

## 相关文档

- [Service 规范](./04-Service规范.md)
- [Response 规范](./12-Response规范.md)
- [DTO 规范](./11-DTO规范.md)
