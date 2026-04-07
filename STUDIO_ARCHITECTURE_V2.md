# Studio 架构重构 V2 - 后端主导设计

## 重构目标

**问题**：旧架构前端需要拼接提示词（prompt），导致业务逻辑泄露到前端。

**解决方案**：后端主导架构 - 前端只传内容和参数，后端根据 YAML skill 配置处理一切。

## 新旧接口对比

### 旧接口（前端拼接提示词）

```bash
POST /api/v1/projects/{id}/materials/studio-html
{
  "title": "我的网页",
  "markdown": "<html>...前端拼接的完整HTML...</html>"
}
```

**问题**：
- 前端需要知道如何构建 HTML
- 提示词模板变更需要改前端
- 不同类型的生成逻辑散落在各处

### 新接口（后端主导）

```bash
POST /api/v1/projects/{id}/studio/create
{
  "type": "html",
  "content": "要展示的文章内容...",
  "title": "我的网页",
  "options": {
    "page_type": "article",
    "theme": "dark",
    "interactive": true
  }
}
```

**优势**：
- 前端只需关注内容和业务参数
- 提示词构建由后端根据 YAML skill 处理
- 新增类型只需添加 YAML 文件，无需改前端

## 架构流程

### 新流程

```
前端请求
  ↓
POST /api/v1/projects/{id}/studio/create
  {
    type: "html",
    content: "...",
    options: {...}
  }
  ↓
后端路由 → HandleProjectStudioCreate
  ↓
加载 Skill YAML (skills/studio/studio-html.yaml)
  ↓
构建提示词 (BuildGenerationPrompt)
  ↓
调用 LLM/TTS 生成
  ↓
保存文件 + 创建 Material
  ↓
返回 {material_id, status, job_id}
```

## API 端点

### 主要接口

| 端点 | 方法 | 说明 |
|------|------|------|
| `POST /api/v1/projects/{id}/studio/create` | POST | **新接口** - 统一的创建入口 |
| `GET /api/v1/studio/skills` | GET | 列出可用的 skills |
| `GET /api/v1/studio/skills/{type}` | GET | 获取 skill 详情 |
| `POST /api/v1/studio/generate` | POST | 通用生成接口（不带 project） |

### 保留的旧接口（向后兼容）

| 端点 | 说明 |
|------|------|
| `POST /api/v1/projects/{id}/materials/slides-pptx` | PPT 生成（旧） |
| `POST /api/v1/projects/{id}/materials/studio-html` | HTML 生成（旧） |
| `POST /api/v1/projects/{id}/materials/studio-audio` | 音频生成（旧） |
| `POST /api/v1/projects/{id}/materials/studio-mindmap` | 思维导图（旧） |

## 新接口详细说明

### 创建生成任务

```bash
POST /api/v1/projects/{id}/studio/create
```

**请求体**：

```typescript
{
  type: "html" | "ppt" | "audio" | "mindmap",
  content: string,        // 要转换的内容
  title?: string,         // 结果标题（可选）
  options?: {
    // HTML 类型
    page_type?: "article" | "report" | "landing" | "slides" | "dashboard" | "interactive",
    theme?: "light" | "dark" | "colorful",
    interactive?: boolean,
    
    // PPT 类型
    slide_count?: number,
    theme?: "professional" | "minimal" | "vibrant" | "tech" | "education",
    include_notes?: boolean,
    
    // Audio 类型
    language?: "zh-CN" | "en-US",
    voice?: string,
    speed?: number,
    
    // Mindmap 类型
    layout?: "radial" | "linear"
  },
  material_id?: number    // 用于更新现有 pending material
}
```

**响应**：

```typescript
{
  success: boolean,
  material_id: number,
  status: "processing" | "done" | "failed",
  job_id: string,
  result: {
    type: string,
    title: string,
    message: string,
    // 类型特定字段
    page_type?: string,
    theme?: string,
    prompt_hash?: string
  }
}
```

### 示例请求

#### 生成 HTML 文章页

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "type": "html",
    "content": "人工智能（AI）是计算机科学的一个分支...",
    "title": "AI 技术介绍",
    "options": {
      "page_type": "article",
      "theme": "light",
      "interactive": true
    }
  }'
```

#### 生成 PPT

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "type": "ppt",
    "content": "## Q4 销售总结\n\n- 营收增长 25%\n- 新客获取 1000+\n- 客户满意度 95%",
    "title": "Q4 销售总结报告",
    "options": {
      "theme": "professional",
      "slide_count": 12,
      "include_notes": true
    }
  }'
```

#### 生成音频

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "type": "audio",
    "content": "欢迎使用 YouMind，您的 AI 知识助手。",
    "title": "欢迎语音",
    "options": {
      "language": "zh-CN",
      "voice": "zh-female-warm",
      "speed": 1.0
    }
  }'
```

## Skill YAML 配置

### 文件位置

```
skills/studio/
├── studio-audio.yaml    # 音频生成配置
├── studio-html.yaml     # HTML 生成配置
└── studio-ppt.yaml      # PPT 生成配置
```

### 配置结构

```yaml
name: studio-html
version: "1.0"
description: "描述"
trigger_keywords:
  - 生成网页
  - HTML
tools:
  - llm_generate
input_schema:
  required: [content]
  optional: [theme, page_type, language]
output_schema:
  fields: [html_url, preview_url, file_size_kb]
theme_options:
  light:
    desc: "明亮主题"
    primary: "#2563eb"
  dark:
    desc: "暗黑主题"  
    primary: "#3b82f6"
---
# Prompt 模板（支持变量替换）
请根据以下内容生成 {{page_type}} 类型的网页：

内容：
{{content}}

主题：{{theme}}
语言：{{language}}

要求：
1. 响应式设计
2. 内联 CSS（不使用外部文件）
3. 语义化 HTML5 标签
```

## 前端迁移指南

### 旧代码

```typescript
// 前端构建提示词
const prompt = buildHTMLPrompt(content, options);
const markdown = await callLLM(prompt);

// 调用接口
await fetch(`/api/v1/projects/${id}/materials/studio-html`, {
  method: 'POST',
  body: JSON.stringify({ title, markdown })
});
```

### 新代码

```typescript
// 直接传参数
const response = await fetch(`/api/v1/projects/${id}/studio/create`, {
  method: 'POST',
  body: JSON.stringify({
    type: 'html',
    content,
    title,
    options
  })
});

const { material_id, job_id } = await response.json();

// 轮询状态
await pollMaterialStatus(material_id);
```

## 后续扩展

要添加新的生成类型：

1. 创建 `skills/studio/studio-xxx.yaml`
2. 在 `HandleProjectStudioCreate` 中添加 case
3. 实现对应的 `handleStudioCreateXXX` 函数
4. 前端无需修改，直接传 `type: "xxx"`

## 实现状态

- ✅ Skill YAML 解析与加载
- ✅ 新接口 `/studio/create`
- ✅ 提示词构建系统
- ✅ 路由分发逻辑
- ✅ 向后兼容的旧接口
- ⏳ LLM 调用集成（等待实现）
- ⏳ 异步任务队列（等待实现）
- ⏳ 状态轮询接口（等待实现）
