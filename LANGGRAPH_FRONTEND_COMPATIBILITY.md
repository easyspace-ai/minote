# LangGraph API 前端兼容性文档

> 本文档列出前端依赖的所有 LangGraph 兼容 API，重构时必须保持这些接口不变。

## 关键 API 端点

### 线程管理
```
GET    /api/threads                    # 列出线
POST   /api/threads                    # 创建线程
POST   /api/threads/search             # 搜索线程
GET    /api/threads/{thread_id}        # 获取线程
PUT    /api/threads/{thread_id}        # 更新线程
PATCH  /api/threads/{thread_id}        # 部分更新
DELETE /api/threads/{thread_id}        # 删除线程
GET    /api/threads/{thread_id}/files  # 获取线程文件
```

**Thread 数据结构**:
```typescript
interface Thread {
  id: string;
  agent_name: string;
  title: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
}
```

### 线程状态
```
GET   /api/threads/{thread_id}/state   # 获取状态
PUT   /api/threads/{thread_id}/state   # 设置状态
POST  /api/threads/{thread_id}/state   # 设置状态
PATCH /api/threads/{thread_id}/state   # 更新状态
```

**State 数据结构**:
```typescript
interface ThreadState {
  values: {
    messages?: Message[];
    title?: string;
    artifacts?: Artifact[];
    todos?: Todo[];
    [key: string]: any;
  };
}
```

### 运行管理
```
GET    /api/threads/{thread_id}/runs              # 列出运行
POST   /api/threads/{thread_id}/runs              # 创建运行
POST   /api/threads/{thread_id}/runs/stream       # 流式运行
GET    /api/threads/{thread_id}/runs/{run_id}     # 获取运行
POST   /api/threads/{thread_id}/runs/{run_id}/stream     # 流式获取运行
POST   /api/threads/{thread_id}/runs/{run_id}/cancel     # 取消运行
POST   /api/threads/{thread_id}/stream            # 加入流
GET    /api/threads/{thread_id}/history           # 获取历史
POST   /api/threads/{thread_id}/history           # 获取历史
```

**Run 数据结构**:
```typescript
interface Run {
  run_id: string;
  thread_id: string;
  status: 'running' | 'success' | 'error';
  created_at: number;
  updated_at: number;
}
```

### SSE 流式事件

前端通过 EventSource 接收以下事件：

```typescript
type StreamEventType = 
  | 'metadata'      // 运行元数据
  | 'chunk'         // 文本片段
  | 'messages'      // 消息更新 [message, metadata]
  | 'tool_call'     // 工具调用开始
  | 'tool_call_start'
  | 'tool_call_end'
  | 'updates'       // 状态更新
  | 'values'        // 完整状态值
  | 'custom'        // 自定义事件（task 等）
  | 'end'           // 运行结束
  | 'error';        // 错误

interface StreamEvent {
  event: StreamEventType;
  data: any;
  id?: string;
}
```

### 消息格式

```typescript
interface Message {
  type: 'human' | 'ai' | 'tool';
  id: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
  additional_kwargs?: Record<string, any>;
  usage_metadata?: {
    input_tokens: number;
    output_tokens: number;
    total_tokens: number;
  };
}

interface ToolCall {
  id: string;
  name: string;
  args: Record<string, any>;
}
```

### 澄清请求
```
GET    /api/threads/{thread_id}/clarifications                    # 列出澄清
POST   /api/threads/{thread_id}/clarifications                    # 创建澄清
GET    /api/threads/{thread_id}/clarifications/{clarification_id} # 获取澄清
POST   /api/threads/{thread_id}/clarifications/{clarification_id}/resolve  # 解决澄清
```

### Agent 管理
```
GET    /api/agents              # 列出 Agent
POST   /api/agents              # 创建 Agent
GET    /api/agents/check        # 检查 Agent 名称
GET    /api/agents/{name}       # 获取 Agent
PUT    /api/agents/{name}       # 更新 Agent
DELETE /api/agents/{name}       # 删除 Agent
```

### Skill 管理
```
GET    /api/skills                    # 列出 Skill
GET    /api/skills/{skill_name}       # 获取 Skill
PUT    /api/skills/{skill_name}       # 启用/禁用 Skill
POST   /api/skills/{skill_name}/enable
POST   /api/skills/{skill_name}/disable
POST   /api/skills/install            # 安装 Skill
```

### 模型管理
```
GET /api/models                 # 列出模型
GET /api/models/{model_name}    # 获取模型详情
```

**Model 数据结构**:
```typescript
interface Model {
  id: string;
  name: string;
  model: string;
  display_name: string;
  description?: string;
  supports_thinking?: boolean;
  supports_reasoning_effort?: boolean;
  supports_vision?: boolean;
  max_tokens?: number;
  temperature?: number;
}
```

### 记忆管理
```
GET    /api/memory                    # 获取记忆
PUT    /api/memory                    # 更新记忆
POST   /api/memory/reload             # 重新加载记忆
DELETE /api/memory                    # 清除记忆
DELETE /api/memory/facts/{fact_id}    # 删除事实
GET    /api/memory/config             # 获取记忆配置
GET    /api/memory/status             # 获取记忆状态
```

### 文件上传
```
POST   /api/uploads                    # 创建上传
GET    /api/uploads/{upload_id}        # 获取上传
PUT    /api/uploads/{upload_id}        # 上传数据
POST   /api/uploads/{upload_id}/complete  # 完成上传
DELETE /api/uploads/{upload_id}        # 删除上传
```

### 其他
```
POST /api/tts                    # 文本转语音
GET  /api/user-profile           # 获取用户配置
PUT  /api/user-profile           # 更新用户配置
GET  /api/channels               # 获取频道列表
POST /api/channels/{name}/restart # 重启频道
GET  /api/mcp/config             # 获取 MCP 配置
PUT  /api/mcp/config             # 更新 MCP 配置
```

## 请求/响应格式

### 创建运行请求
```typescript
interface RunCreateRequest {
  assistant_id?: string;
  input?: {
    messages?: Array<{
      role: 'human' | 'user' | 'ai' | 'assistant';
      content: string | Array<{type: 'text'; text: string} | {type: 'image_url'; image_url: {url: string}}>;
    }>;
  };
  config?: {
    configurable?: {
      model_name?: string;
      agent_name?: string;
      agent_type?: string;
      reasoning_effort?: string;
      thinking_enabled?: boolean;
      is_plan_mode?: boolean;
      subagent_enabled?: boolean;
      max_concurrent_subagents?: number;
      [key: string]: any;
    };
  };
  context?: Record<string, any>;
  feedback?: string;
  auto_accepted_plan?: boolean;
}
```

## 前端使用示例

### 创建线程并运行
```typescript
// 1. 创建线程
const thread = await fetch('/api/threads', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ agent_name: 'default' })
}).then(r => r.json());

// 2. 启动流式运行
const eventSource = new EventSource(
  `/api/threads/${thread.id}/runs/stream`,
  { headers: { 'Content-Type': 'application/json' } }
);

eventSource.addEventListener('metadata', (e) => {
  const data = JSON.parse(e.data);
  console.log('Run started:', data.run_id);
});

eventSource.addEventListener('chunk', (e) => {
  const data = JSON.parse(e.data);
  console.log('Text:', data.delta);
});

eventSource.addEventListener('tool_call', (e) => {
  const data = JSON.parse(e.data);
  console.log('Tool called:', data.name);
});

eventSource.addEventListener('end', (e) => {
  console.log('Run completed');
  eventSource.close();
});

eventSource.addEventListener('error', (e) => {
  console.error('Error:', e);
});
```

## 兼容性检查清单

重构时必须验证：

- [ ] 所有端点返回正确的 HTTP 状态码
- [ ] Thread 数据结构字段完整
- [ ] Run 数据结构字段完整
- [ ] Message 数据结构字段完整
- [ ] SSE 事件类型完整
- [ ] 流式响应正常工作
- [ ] 工具调用事件正常
- [ ] 上传功能正常
- [ ] 历史记录获取正常
- [ ] 状态管理正常

## 测试命令

```bash
# 启动服务
go run ./cmd/gateway

# 运行后端测试
go test ./pkg/langgraphcompat/... -v

# 前端测试（在前端目录）
cd web && pnpm test
```
