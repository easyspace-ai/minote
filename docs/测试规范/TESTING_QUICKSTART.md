# YouMind 测试代码快速入门

## 已创建的测试文件

```
youmind/
├── cmd/                      # 命令入口
├── internal/                 # 内部实现
│   ├── server/              # HTTP 服务
│   ├── workflow/            # 工作流引擎
│   └── ...
├── pkg/                      # 可复用包
│   ├── agent/               # Agent 框架 (Eino)
│   │   └── *_test.go
│   └── tools/               # 工具集
│       └── *_test.go
├── web/                      # React 19 前端
│   ├── src/
│   │   ├── components/      # 组件测试
│   │   ├── hooks/           # Hook 测试
│   │   └── services/        # API 服务测试
│   └── package.json
└── skills/                   # AI Skills
    └── */SKILL.md
```

---

## 后端测试运行指南

### 1. 启动依赖服务

```bash
# 启动基础设施（Postgres, Redis, MarkItDown, MinIO 等）
make infra

# 开发模式启动（热重载）
make dev-notex
```

### 2. 运行所有测试

```bash
# Go 后端测试
go test -v ./...

# 特定包测试
go test -v ./pkg/agent/...        # Agent 框架测试
go test -v ./pkg/tools/...        # 工具集测试
go test -v ./internal/workflow/... # 工作流测试
```

### 3. 运行特定测试函数

```bash
# 运行单个测试
go test -v ./pkg/agent -run TestReactAgent

# 运行相关测试
go test -v ./pkg/tools -run TestToolRegistry

# 使用短模式跳过慢速测试
go test -v -short ./...
```

### 4. 生成测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率
go tool cover -func=coverage.out

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```

### 5. 并发/竞态检测

```bash
# 使用 race detector 运行测试
go test -race ./...
```

---

## 前端测试运行指南

### 1. 安装依赖

```bash
cd web
pnpm install
```

### 2. 运行测试

```bash
# TypeScript 类型检查
pnpm typecheck

# 运行所有测试
pnpm test

# 运行测试（watch 模式）
pnpm test -- --watch

# 运行测试并显示 UI
pnpm test:ui

# 运行特定测试文件
pnpm test Button.test.tsx
```

### 3. 生成覆盖率

```bash
# 生成覆盖率报告
pnpm test:coverage
```

---

## 测试统计

### 后端测试 (Go 1.25 + Eino)

| 层级 | 测试文件 | 说明 |
|------|---------|------|
| pkg/agent | *_test.go | Agent 框架、ReAct 循环、SubAgent |
| pkg/tools | *_test.go | 工具注册表、内置工具、路径访问控制 |
| internal | *_test.go | 业务逻辑、工作流、HTTP 处理 |

### 前端测试 (React 19)

| 类型 | 说明 |
|------|------|
| 组件测试 | React 组件渲染与交互 |
| Hook 测试 | 自定义 Hooks 逻辑 |
| 服务测试 | API 调用、数据转换 |

---

## AI 生成测试模板

### 后端测试模板

```go
package {package_name}

import (
    "testing"
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)

// Mock{RepositoryName} Mock仓库接口
type Mock{RepositoryName} struct {
    mock.Mock
}

// Test{ServiceName}_{Action} 测试说明
func Test{ServiceName}_{Action}(t *testing.T) {
    tests := []struct {
        name        string
        input       interface{}
        mockSetup   func(*Mock{RepositoryName})
        expected    interface{}
        expectedErr error
    }{
        {
            name: "正常场景",
            input: {input_value},
            mockSetup: func(m *Mock{RepositoryName}) {
                m.On("MethodName", mock.Anything, {args}).
                    Return({result}, nil)
            },
            expected: {expected_result},
        },
        {
            name: "错误场景",
            input: {error_input},
            mockSetup: func(m *Mock{RepositoryName}) {
                m.On("MethodName", mock.Anything, {args}).
                    Return(nil, {error})
            },
            expectedErr: {error},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(Mock{RepositoryName})
            tt.mockSetup(mockRepo)

            service := New{ServiceName}(mockRepo)
            result, err := service.Method(context.Background(), tt.input)

            if tt.expectedErr != nil {
                assert.Error(t, err)
                assert.Equal(t, tt.expectedErr, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### 前端测试模板

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { rest } from 'msw'
import { setupServer } from 'msw/node'
import userEvent from '@testing-library/user-event'
import {ComponentName} from './{ComponentName}'

const server = setupServer()

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('{ComponentName}', () => {
    beforeEach(() => {
        server.use(
            rest.get('/api/endpoint', (req, res, ctx) => {
                return res(ctx.status(200), ctx.json({ data: 'mock' }))
            })
        )
    })

    describe('渲染测试', () => {
        it('应该正确渲染组件', async () => {
            render(<ComponentName />)

            await waitFor(() => {
                expect(screen.getByText('Expected Text')).toBeInTheDocument()
            })
        })
    })

    describe('交互测试', () => {
        it('应该响应用户操作', async () => {
            const user = userEvent.setup()
            render(<ComponentName />)

            await user.click(screen.getByRole('button', { name: /action/i }))

            await waitFor(() => {
                expect(screen.getByText('Result')).toBeInTheDocument()
            })
        })
    })
})
```

---

## 常见问题

### Q1: 后端测试报错 "undefined: testify"

**A**: 确保 go.mod 中包含 testify 依赖：
```bash
go get github.com/stretchr/testify
```

### Q2: 前端测试报错 "MSW is not configured"

**A**: 确保在测试文件顶部有 MSW server 的 setup：
```typescript
const server = setupServer()
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

### Q3: 测试运行很慢

**A**: 后端可以使用 `-short` flag 跳过慢速测试：
```bash
go test -v -short ./...
```

前端可以运行特定测试：
```bash
pnpm test {test_file}
```

---

## 下一步

1. 运行测试，验证测试代码的正确性
2. 根据测试结果调整测试代码
3. 将测试代码合并回主分支
4. 为其他功能模块创建类似的测试

---

## 相关文档

- 测试规范目录：`docs/测试规范/`
- 后端测试规范：[后端自动化测试规范.md](./后端自动化测试规范.md)
- 前端测试规范：[前端自动化测试规范.md](./前端自动化测试规范.md)
- 测试模板：上述代码模板
