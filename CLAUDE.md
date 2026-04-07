# CLAUDE.md — YouMind / Minote AI 协作规范

> **必读顺序**: `CLAUDE.md`（本文件）→ `AGENTS.md`（架构细节）→ 相关 `skills/*.yaml`

---

## 核心原则

| 原则 | 说明 |
|------|------|
| 先理解后编码 | 必须先阅读相关代码和文档，再动手 |
| 小批量提交 | 每次修改范围聚焦，易于回滚和审查 |
| 测试先行 | 新功能必须有测试；重构必须保持测试通过 |
| 文档同步 | 代码变更同步更新 AGENTS.md / skill YAML |
| 单一职责 | 每个 skill、tool、handler 只做一件事 |

---

## 开始工作前的必检清单

```
□ 已读 AGENTS.md（架构、模块边界、约定）
□ 已理解相关代码的数据流向
□ 确认环境变量 / config.yaml 配置正确
□ 基础设施正在运行（docker compose ps）
□ 知道如何运行受影响模块的测试
```

### 环境快速检查

```bash
# 检查基础设施
docker compose ps

# 若未启动
make infra

# 检查配置
cat config.yaml

# 编译验证（任何改动后都应执行）
go build ./...
```

---

## 开发工作流

### 1. 理解需求（禁止跳过）

向用户确认：
- 功能目标 & 验收标准
- 涉及哪些模块（参见 AGENTS.md 目录结构）
- 是否有设计文档 / PRD / 参考实现
- 优先级和截止时间

**禁止**：未确认需求就开始编码。

### 2. 探索代码

```bash
# 定位相关代码
grep -r "关键词" --include="*.go" pkg/ internal/

# 查看 skill 定义
ls skills/
cat skills/<skill-name>.yaml

# 查看工具注册
cat pkg/tools/registry.go
```

### 3. 编写代码规范

#### Go 后端

```go
// ✅ 注释解释"为什么"，不解释"是什么"
// retryOnRateLimit wraps LLM calls with exponential backoff because
// upstream model APIs return 429 transiently under burst traffic.
func retryOnRateLimit(ctx context.Context, fn func() error) error { ... }

// ✅ 错误处理：具体错误码 + 结构化日志
func (s *Server) handleStudioGenerate(w http.ResponseWriter, r *http.Request) {
    var req StudioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeAPIError(w, http.StatusBadRequest,
            NewAPIError(ErrCodeInvalidRequest, "invalid JSON body"))
        return
    }
    result, err := s.studioService.Generate(r.Context(), req)
    if err != nil {
        s.logger.Printf("[ERROR] studio generate failed: type=%s err=%v", req.Type, err)
        writeAPIError(w, http.StatusInternalServerError,
            NewAPIError(ErrCodeInternal, "generation failed"))
        return
    }
    writeJSON(w, http.StatusOK, result)
}

// ✅ 用 Options struct 代替多参数
type GenerateOptions struct {
    Type     StudioType // audio | html | ppt
    Input    string
    Language string
    Voice    string // audio only
    Theme    string // ppt only
}
```

#### 前端 (React 19 + TypeScript)

```tsx
// ✅ 严格类型 — 禁止 any
interface StudioCardProps {
  job: StudioJob;
  onRetry: (id: string) => void;
  onDownload: (id: string, format: string) => void;
}

// ✅ 函数式组件 + 解构
export function StudioCard({ job, onRetry, onDownload }: StudioCardProps) { ... }

// ✅ TanStack Query 管理异步状态
function useStudioJobs(type: StudioType) {
  return useQuery({
    queryKey: ['studio-jobs', type],
    queryFn: () => fetchStudioJobs(type),
    staleTime: 30_000,
  });
}
```

#### Skill YAML 规范

每个 Studio skill 必须遵循独立文件结构（见 skills/ 目录）：

```yaml
name: studio-<type>           # audio | html | ppt
version: "1.0"
description: "单行说明，清楚触发条件"
trigger_keywords:             # 触发关键词列表
  - keyword1
tools:                        # 声明依赖工具
  - tool_name
input_schema:                 # 明确输入契约
  required: [field1]
  optional: [field2]
output_schema:                # 明确输出契约
  fields: [...]
prompt: |
  # 角色 & 目标
  # 输入处理规则
  # 输出格式要求
  # 错误处理
examples:
  - input: "..."
    output: "..."
```

---

## 常见任务指引

### 添加新 API 端点

```
1. internal/notex/server_core.go  ← 注册路由
2. internal/notex/*_handler.go    ← 实现 Handler（每类资源独立文件）
3. internal/notex/*_domain.go     ← 业务逻辑
4. internal/notex/types.go        ← 数据模型（如需）
5. *_test.go                      ← 测试
```

### 添加新 Tool

```
1. pkg/tools/builtin/<toolname>.go    ← 实现
2. pkg/tools/registry.go             ← 注册
3. pkg/tools/builtin/<toolname>_test.go
```

### 添加新 Studio Skill（重点）

Studio 的 audio / html / ppt 生成**必须**拆分为独立 skill 文件：

```
skills/
├── studio-audio.yaml    ← TTS / 音频生成
├── studio-html.yaml     ← 交互式 HTML 生成
└── studio-ppt.yaml      ← PowerPoint 生成
```

每个 skill 独立、自描述、可单独测试。**禁止**将多种生成类型合并到一个 skill 或 handler 中。

### 修改数据模型

```
1. internal/notex/types.go       ← 修改结构体
2. GORM AutoMigrate 自动处理新字段
3. 复杂迁移 → 独立脚本 scripts/migrate_*.go
4. 更新相关 Handler 和序列化逻辑
```

---

## 代码审查清单（提交前必过）

```
□ go build ./...           ✓ 编译通过
□ go test ./...            ✓ 所有测试通过
□ go test -race ./...      ✓ 无数据竞争
□ go vet ./...             ✓ 静态检查
□ cd web && pnpm typecheck ✓ 前端类型安全
□ 错误路径有日志且有单元测试
□ 新 Skill 有 examples 字段
□ AGENTS.md 已同步更新（若有架构变动）
□ 无硬编码 API Key / 密码 / Secret
```

---

## Studio 模块重构要点

当前问题（已知）：
- audio / html / ppt 生成逻辑耦合在同一处理器或 skill 中
- 缺少独立的输入输出 schema 约束
- 错误处理不统一
- 无法单独扩展或替换某类生成能力

重构方向：
- 每种生成类型 → 独立 skill YAML（详见 skills/ 目录）
- 后端每种类型 → 独立 Handler + Service 方法
- 共享的 LLM 调用逻辑 → `pkg/studio/` 公共包
- 统一的任务状态模型（pending → processing → done / failed）

---

## 调试

### 后端

```bash
# 详细日志
export LOG_LEVEL=debug
go run ./cmd/notex

# Delve 调试
dlv debug ./cmd/notex -- --port 8787
```

### 前端

```bash
cd web && pnpm dev
# 修改 vite.config.ts proxy 指向本地后端
```

### 常见问题速查

| 症状 | 排查 |
|------|------|
| 服务启动失败 | `docker compose ps` — Postgres/Redis/MinIO 是否健康 |
| 401 Unauthorized | Authorization header 格式 / JWT 是否过期 |
| LLM 调用失败 | `config.yaml` API Key / model 名称 / base_url |
| Studio 生成卡住 | 检查 TTS / volc-tts 服务状态和 token 配额 |
| 前端 404 | Gateway 是否运行，代理路径是否匹配 |

---

## 安全红线

```
✗ 禁止提交 API Key、密码、JWT Secret 到 git
✗ 禁止拼接 SQL 字符串（使用 GORM 参数化）
✗ 禁止在前端直接暴露 LLM API Key
✓ 用户输入必须验证长度和格式
✓ 文件上传必须校验 MIME type 和大小
```

---

## 提交规范

```
<type>(<scope>): <subject>

[body — 可选，解释"为什么"]
```

**type**: `feat` | `fix` | `refactor` | `test` | `docs` | `chore`  
**scope 示例**: `studio` | `notex` | `agent` | `skill` | `llm` | `web`

```
# 示例
feat(studio): split audio/html/ppt into independent skills

- Add skills/studio-audio.yaml with TTS options and voice selection
- Add skills/studio-html.yaml with template and interactivity modes
- Add skills/studio-ppt.yaml with slide structure and theme support
- Remove monolithic studio skill that coupled all three types
```

---

## 沟通原则

1. **不确定就问** — 需求模糊时立刻询问，不猜测
2. **解释决策** — "选择 X 而非 Y，因为…"
3. **提供选项** — 给出至少两个方案 + 推荐理由
4. **提前预警** — 发现设计问题立刻提出，不等到完成后

---

> **记住**: 每个 Studio skill 都是独立的能力单元。可扩展性 > 短期便利。
