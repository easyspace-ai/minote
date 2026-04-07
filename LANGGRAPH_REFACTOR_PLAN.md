# LangGraph 兼容层重构计划

## 目标
将 `pkg/langgraphcompat/gateway.go`（4042 行）重构为模块化结构，同时保持 API 兼容性。

## 现状分析

### 文件结构
```
pkg/langgraphcompat/
├── gateway.go           # 4042 行 - 需要拆分的主文件
├── handlers.go          # 1980 行 - 部分处理逻辑已拆分
├── threads.go           # 线程管理
├── compat.go            # 兼容层
├── types/               # 新创建的类型包
│   └── types.go         # 共享类型定义
└── utils/               # 新创建的工具包
    ├── utils.go         # 工具函数
    └── errors.go        # 错误定义
```

### API 端点统计
- `/api/tts` - TTS 服务
- `/api/models` (2) - 模型管理
- `/api/skills` (6) - Skill 管理
- `/api/agents` (6) - Agent 管理
- `/api/user-profile` (2) - 用户配置
- `/api/memory` (8) - 记忆管理
- `/api/channels` (2) - 频道管理
- `/api/mcp/config` (2) - MCP 配置
- `/api/threads/*` (20+) - 线程/运行管理
- `/api/uploads/*` (5) - 文件上传

总计：约 50+ 个端点

## 重构策略

### 阶段 1：类型和工具提取（已完成）
- ✅ 创建 `types/` 包 - 共享类型定义
- ✅ 创建 `utils/` 包 - 工具函数和错误定义

### 阶段 2：按领域拆分 Handler（推荐）
创建以下 handler 文件：

```
pkg/langgraphcompat/
├── handlers/
│   ├── thread_handler.go      # 线程管理（从 threads.go 迁移）
│   ├── skill_handler.go       # Skill 管理（从 gateway.go 提取）
│   ├── agent_handler.go       # Agent 管理（从 gateway.go 提取）
│   ├── memory_handler.go      # 记忆管理（从 gateway.go 提取）
│   ├── model_handler.go       # 模型管理（从 gateway.go 提取）
│   ├── upload_handler.go      # 上传管理（从 upload_conversion.go 迁移）
│   └── run_handler.go         # 运行管理（从 handlers.go 迁移）
```

### 阶段 3：创建模块化 Gateway（推荐）
创建 `gateway_v2.go`：

```go
type ModularGateway struct {
    // 依赖注入
    llm          llm.LLMProvider
    threadStore  ThreadStore
    
    // Handler 组合
    threadHandler *ThreadHandler
    skillHandler  *SkillHandler
    agentHandler  *AgentHandler
    // ...
}
```

## 前端兼容性保证

### API 契约保持不变
所有重构必须保持：
1. **URL 路径** - 不改变任何端点路径
2. **请求/响应格式** - JSON 结构完全一致
3. **状态码** - HTTP 状态码语义一致
4. **SSE 事件格式** - 流式响应格式一致
5. **错误格式** - 错误响应结构一致

### 关键数据结构
```typescript
// Thread 结构 - 必须保持一致
interface Thread {
  id: string;
  agent_name: string;
  title: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
}

// Run 结构 - 必须保持一致
interface Run {
  run_id: string;
  thread_id: string;
  status: 'running' | 'success' | 'error';
  created_at: number;
  updated_at: number;
}

// SSE 事件 - 必须保持一致
interface StreamEvent {
  event: 'metadata' | 'chunk' | 'tool_call' | 'end' | 'error';
  data: any;
  id?: string;
}
```

## 迁移步骤

### 第 1 步：创建测试覆盖
```bash
# 为现有 API 创建集成测试
go test -v ./pkg/langgraphcompat/... -run TestAPI
```

### 第 2 步：逐模块迁移
每次迁移一个 handler：
1. 创建新 handler 结构
2. 复制并整理代码
3. 更新路由注册
4. 运行测试验证

### 第 3 步：验证前端兼容性
```bash
# 启动后端
go run ./cmd/gateway

# 运行前端测试（在前端目录）
cd web && pnpm test

# 手动验证关键流程
# 1. 创建线程
# 2. 发送消息
# 3. 接收流式响应
# 4. 查看历史记录
```

## 风险控制

### 回滚策略
1. 保留 `gateway.go` 直到重构完成
2. 使用功能开关切换新旧实现
3. 保持数据库 schema 兼容

### 测试策略
1. 单元测试 - 每个 handler 独立测试
2. 集成测试 - API 端点完整测试
3. 端到端测试 - 前端-后端联合测试

## 实施建议

### 短期（本周）
1. 完成类型和工具包提取 ✅
2. 编写 API 契约文档
3. 创建集成测试基线

### 中期（2-4 周）
1. 逐个迁移 handler（每周 2-3 个）
2. 保持 API 兼容性测试通过
3. 性能基准测试对比

### 长期（1-2 月）
1. 完整迁移完成后移除旧代码
2. 性能优化
3. 代码审查和文档更新

## 当前状态

已在分支 `refactor/langgraph-modular` 上完成：
- ✅ 创建 `types/types.go` - 类型定义
- ✅ 创建 `utils/utils.go` - 工具函数
- ✅ 创建 `utils/errors.go` - 错误定义
- ✅ 创建目录结构 `handlers/`, `middleware/`, `transform/`

下一步建议：
1. 编写集成测试建立基线
2. 选择第一个 handler 进行迁移（建议从 model_handler 开始，最简单）
3. 验证前端兼容性
