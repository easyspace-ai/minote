# AGENTS.md — YouMind / Minote 架构与开发指南

> 本文档为 AI Agent 提供完整的项目架构、模块边界、约定和决策依据。  
> 配合 `CLAUDE.md` 使用。有代码问题时，优先查本文档。

---

## 项目定位

**YouMind (Minote)** 是基于 **Go + CloudWeGo Eino** 的 AI 知识管理与创作平台，核心能力：

- 智能对话（多轮 / 工具调用 / 流式）
- 知识库管理（文库、文档、向量检索）
- **Studio 创作**（音频 TTS、HTML 页面、PPT 演示文稿）
- AI Agent 编排（ReAct / 子 Agent / Skills 系统）
- 代码仓库智能解读（openDeepWiki 功能）

**模块路径**: `github.com/easyspace-ai/minote`  
**Go 版本**: 1.25+

---

## 系统架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (React 19 + Vite)                  │
│            Tailwind CSS 4 · Radix UI · shadcn/ui               │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP / SSE / WebSocket
┌──────────────────────────▼──────────────────────────────────────┐
│                      Gateway (:8080)                            │
│          静态资源服务 · API 路由转发 · 认证中间件               │
└──────┬───────────────────┬────────────────────────┬────────────┘
       │                   │                        │
┌──────▼──────┐   ┌────────▼────────┐   ┌──────────▼──────────┐
│   Notex     │   │   Agent 服务     │   │   Studio 服务        │
│  (:8787)    │   │  (Eino ReAct)   │   │  audio/html/ppt      │
│  核心业务   │   │  工具调用/子Agent │   │  TTS · 生成 · 渲染   │
└──────┬──────┘   └────────┬────────┘   └──────────┬──────────┘
       │                   │                        │
       └───────────────────┴────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                     共享基础设施层                               │
│  PostgreSQL · Redis · MinIO · Milvus · Neo4j · MarkItDown       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 服务目录 (cmd/)

| 服务 | 端口 | 职责 | 状态 |
|------|------|------|------|
| `gateway` | :8080 | 统一网关，路由转发，静态资源 | 核心 |
| `notex` | :8787 | 用户、项目、文档、对话、知识库 | 核心 |
| `agent` | — | Agent 编排执行（Eino） | 核心 |
| `langgraph` | — | LangGraph 兼容 API | 核心 |
| `checkpoint` | — | 对话状态持久化 | 核心 |
| `memory` | — | 向量存储与语义记忆 | 核心 |
| `tts` | — | 标准 TTS 音频合成 | Studio |
| `volc-tts` | — | 火山引擎 TTS（高质量） | Studio |

---

## 目录结构（详解）

```
youmind/
├── cmd/                          # 服务入口（每个子目录=独立可执行）
│   ├── gateway/
│   ├── notex/
│   ├── agent/
│   ├── langgraph/
│   ├── checkpoint/
│   ├── memory/
│   ├── tts/                      # 标准 TTS
│   └── volc-tts/                 # 火山引擎 TTS
│
├── internal/notex/               # Notex 核心业务（禁止被其他包 import）
│   ├── server_core.go            # HTTP 路由注册
│   ├── types.go                  # 所有领域模型（GORM 结构体）
│   ├── *_handler.go              # HTTP 处理器（每类资源一个文件）
│   ├── *_domain.go               # 业务逻辑（无 HTTP 依赖）
│   ├── store*.go                 # 数据访问层
│   └── notexapp/                 # 依赖注入 / 应用组装
│
├── pkg/                          # 共享包（可被多服务 import）
│   ├── agent/                    # Eino ReAct Agent 实现
│   ├── llm/                      # LLM 客户端、多模型支持、token 追踪
│   ├── tools/                    # 工具注册与实现
│   │   ├── registry.go           # 工具注册中心
│   │   ├── skills.go             # Skill 加载与调度
│   │   └── builtin/              # 内置工具（bash, web_search, file_read...）
│   ├── studio/                   # ★ Studio 生成公共逻辑（重构目标）
│   │   ├── audio.go              # TTS 调用封装
│   │   ├── html.go               # HTML 生成封装
│   │   ├── ppt.go                # PPT 生成封装
│   │   └── types.go              # 共享类型（StudioJob, StudioType...）
│   ├── langgraphcompat/          # LangGraph API 兼容层
│   ├── checkpoint/               # Checkpoint 存储
│   ├── memory/                   # 记忆管理
│   ├── subagent/                 # 子 Agent 执行
│   ├── mcp/                      # MCP 客户端（OAuth + 工具发现）
│   ├── cache/                    # 缓存抽象
│   └── gateway/                  # 网关共享代码
│
├── skills/                       # ★ Skill 定义（YAML，Agent 可发现）
│   ├── studio-audio.yaml         # 音频生成 skill
│   ├── studio-html.yaml          # HTML 生成 skill
│   ├── studio-ppt.yaml           # PPT 生成 skill
│   └── *.yaml                    # 其他 skill
│
├── web/src/
│   ├── components/
│   │   ├── ui/                   # shadcn 基础组件
│   │   ├── ai-elements/          # AI 特有组件
│   │   ├── workspace/            # 工作区组件
│   │   └── studio/               # ★ Studio 相关组件（按类型拆分）
│   │       ├── AudioStudio.tsx
│   │       ├── HtmlStudio.tsx
│   │       └── PptStudio.tsx
│   ├── pages/
│   └── core/                     # i18n、任务调度
│
├── agents/                       # Agent YAML 配置（运行时释放）
├── docker/                       # Dockerfile 集合
├── scripts/                      # 迁移脚本、工具脚本
├── config.yaml                   # 主配置
├── docker-compose.yml
└── Makefile
```

---

## 技术栈

### 后端

| 组件 | 技术 | 备注 |
|------|------|------|
| 语言 | Go 1.25+ | 泛型 / 标准库路由 |
| LLM 框架 | CloudWeGo Eino | ReAct / Graph / Tool |
| HTTP | `net/http` 标准库 | Go 1.22+ ServeMux |
| 数据库 | PostgreSQL (ParadeDB) | 全文检索 |
| ORM | GORM | AutoMigrate |
| 缓存 | Redis (go-redis/v9) | |
| 向量 DB | Milvus | 语义检索 |
| 图数据库 | Neo4j | 知识图谱 |
| 对象存储 | MinIO | 文件、图片 |
| 文档解析 | MarkItDown (gRPC) | PDF → Markdown |
| CLI | Cobra | |

### 前端

| 组件 | 技术 |
|------|------|
| 框架 | React 19 |
| 语言 | TypeScript 5.8 |
| 构建 | Vite 6 |
| 样式 | Tailwind CSS 4 |
| 组件 | Radix UI + shadcn/ui |
| 路由 | React Router 7 |
| 数据 | TanStack Query v5 |
| 代码编辑 | CodeMirror 6 |
| Markdown | react-markdown + remark/rehype |
| LangGraph SDK | @langchain/langgraph-sdk |

---

## Studio 模块设计（重构后）

### 问题诊断（重构前）

```
❌ audio / html / ppt 生成逻辑混在同一 handler/skill
❌ 无统一的任务状态机（pending→processing→done/failed）
❌ 无输入输出 schema 约束，难以前端对接
❌ TTS 服务（tts / volc-tts）调用方式不一致
❌ 错误信息不结构化，难以前端展示
```

### 重构后的设计原则

```
✓ 每种类型 → 独立 skill YAML（skills/studio-*.yaml）
✓ 每种类型 → 独立 Handler（internal/notex/studio_*_handler.go）
✓ 共享业务逻辑 → pkg/studio/（audio.go / html.go / ppt.go）
✓ 统一任务模型 → StudioJob{ID, Type, Status, Input, Output, Error}
✓ 统一 API → POST /api/v1/studio/:type/generate
✓ 状态轮询 → GET /api/v1/studio/jobs/:id
✓ 流式输出 → GET /api/v1/studio/jobs/:id/stream (SSE)
```

### Studio 任务状态机

```
[pending] ──► [processing] ──► [done]
                    │
                    └──► [failed] ──► [pending] (retry)
```

### 统一数据模型

```go
// pkg/studio/types.go
type StudioType string
const (
    StudioTypeAudio StudioType = "audio"
    StudioTypeHTML  StudioType = "html"
    StudioTypePPT   StudioType = "ppt"
)

type StudioJob struct {
    ID        string     `json:"id" gorm:"primaryKey"`
    Type      StudioType `json:"type"`
    Status    string     `json:"status"` // pending|processing|done|failed
    Input     string     `json:"input"`  // JSON-encoded input params
    Output    string     `json:"output"` // URL / base64 / HTML string
    Error     string     `json:"error,omitempty"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

---

## 关键约定

### API 设计

- 路由使用 Go 1.22+ `http.NewServeMux()`
- 路径格式: `/api/v1/{resource}/{id}/{action}`
- 中间件顺序: `RequestID → Logging → CORS → Auth → Recovery`
- 统一响应结构:

```go
type APIResponse struct {
    Data  any       `json:"data,omitempty"`
    Error *APIError `json:"error,omitempty"`
}
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### Skill 约定

- 每个 skill = `skills/<name>.yaml`，自描述，包含 trigger_keywords / input_schema / output_schema / examples
- Skill 扫描由 `pkg/tools/skills.go` 自动发现
- Studio skill 之间**不可互相依赖**，可复用 `tools` 字段声明共享工具

### 错误处理

```go
// 定义语义错误
var (
    ErrStudioInputTooLong  = errors.New("studio: input exceeds limit")
    ErrStudioTTSUnavailable = errors.New("studio: TTS service unavailable")
)

// HTTP 层转换
func mapStudioError(err error) (int, *APIError) {
    switch {
    case errors.Is(err, ErrStudioInputTooLong):
        return http.StatusBadRequest, NewAPIError("STUDIO_INPUT_TOO_LONG", err.Error())
    case errors.Is(err, ErrStudioTTSUnavailable):
        return http.StatusServiceUnavailable, NewAPIError("STUDIO_TTS_UNAVAILABLE", err.Error())
    default:
        return http.StatusInternalServerError, NewAPIError("INTERNAL", "internal error")
    }
}
```

### 并发安全

- 细粒度锁：按领域划分 `sync.RWMutex`
- ID 生成用 `atomic.AddInt64` 或 `uuid.New()`
- Studio 任务队列考虑使用 Redis List 或 channel（避免内存丢失）

### 配置优先级

```
环境变量 > config.yaml > 默认值
```

关键环境变量：

```bash
POSTGRES_URL        # 数据库
REDIS_ADDR          # Redis
REDIS_PASSWORD
OPENAI_API_KEY      # LLM
ANTHROPIC_API_KEY
SILICONFLOW_API_KEY
MILVUS_ADDRESS      # 向量库
VOLC_TTS_ACCESS_KEY # 火山 TTS
VOLC_TTS_SECRET_KEY
NOTEX_DATA_ROOT     # 数据目录
```

---

## 开发命令速查

```bash
# 基础设施
make infra                  # 启动 Postgres/Redis/MinIO 等
make dev-notex              # 热重载开发（Notex）
make down                   # 停止所有

# 测试
go test ./...               # 全量
go test ./pkg/studio/...    # Studio 模块
go test -race ./...         # 竞态检测
go test -cover ./...        # 覆盖率

# 前端
cd web && pnpm install
cd web && pnpm dev
cd web && pnpm typecheck
cd web && pnpm build

# 构建
go build ./cmd/notex
make notex-build            # Docker 镜像
```

---

## Agent 系统

### 工具注册流程

```
pkg/tools/builtin/<tool>.go
    └─► pkg/tools/registry.go  (Register)
        └─► pkg/agent/         (注入 Eino Agent)
            └─► skills/*.yaml  (声明使用)
```

### Skill 加载流程

```
1. 启动时扫描 skills/ 目录
2. 解析 YAML → SkillDefinition
3. 注入 Agent 的工具列表
4. 用户消息命中 trigger_keywords → 激活 skill
```

### 子 Agent

- `pkg/subagent/` 封装子 Agent 调用
- 支持并行执行多个子 Agent（Go routines + channel 收集结果）
- 超时 / 取消通过 `context.Context` 控制

---

## 数据库约定

- 模型定义统一在 `internal/notex/types.go`
- GORM AutoMigrate 处理字段新增
- 复杂数据迁移 → `scripts/migrate_*.go` 独立脚本
- 禁止在 Handler 层直接操作数据库；通过 `store*.go` 访问

---

## 部署

```bash
# Docker Compose（推荐）
docker compose --profile app up -d
docker compose ps

# 二进制
go build -o notex ./cmd/notex
./notex
```

---

## 参考资源

- [CloudWeGo Eino](https://github.com/cloudwego/eino)
- [LangGraph 协议](https://langchain-ai.github.io/langgraph/)
- [ParadeDB 全文检索](https://docs.paradedb.com/)
- [Tailwind CSS 4](https://tailwindcss.com/)
- [Radix UI](https://www.radix-ui.com/)
- [TanStack Query](https://tanstack.com/query/latest)
- [火山引擎 TTS](https://www.volcengine.com/product/tts)
