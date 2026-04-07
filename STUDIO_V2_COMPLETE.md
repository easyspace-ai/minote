# Studio 创作系统 V2 - 完整更新文档

## 🎯 核心改进

### 1. 架构重构：后端主导

**旧架构**：前端拼接提示词 → 调用 AI → 再调用具体接口
**新架构**：前端只传参数 → 后端构建提示词 → 后端直接生成

```
前端请求: { type: "ppt", content: "...", options: { theme: "professional" } }
    ↓
后端路由 → 加载 Skill YAML → 构建提示词 → LLM 生成 → 文件保存
    ↓
返回: { material_id, status: "ready", download_url }
```

### 2. PPT 生成重构（基于 PPT-as-code 设计）

**问题修复**：
- ❌ 旧版：纯色文字、无布局、无颜色
- ✅ 新版：5 种专业主题、彩色设计、清晰布局

**支持的主题**：
| 主题 | 描述 | 主色 |
|------|------|------|
| professional | 专业商务 | 深蓝 #1e3a5f |
| minimal | 极简风格 | 黑白灰 |
| vibrant | 活力创意 | 紫青渐变 |
| tech | 科技感 | 深色+霓虹 |
| education | 教育培训 | 温暖色调 |

**幻灯片布局**：
- 封面页（标题+副标题+装饰条）
- 内容页（标题栏+要点列表）
- 章节分隔页（大标题+主题色背景）
- 结束页（感谢语）

## 📁 新增文件

### 后端

| 文件 | 说明 |
|------|------|
| `internal/notex/studio_handlers_v2.go` | 新的统一创建入口 |
| `internal/notex/studio_skill_integration.go` | Skill 系统核心 |
| `internal/notex/pptx_generator_v2.go` | PPTX 主题生成器 |

### Skill YAML（skills/studio/）

| 文件 | 类型 | 说明 |
|------|------|------|
| `studio-audio.yaml` | 音频 | TTS 语音合成 |
| `studio-html.yaml` | HTML | 网页生成 |
| `studio-ppt.yaml` | PPT | 演示文稿 |
| `studio-infographic.yaml` | 信息图 | 数据可视化 |
| `studio-quiz.yaml` | 测验 | 互动问答 |
| `studio-data-table.yaml` | 数据表 | 表格生成 |

### 前端

| 文件 | 说明 |
|------|------|
| `web/src/lib/studioCreateV2.ts` | 新的 API 调用库 |

## 🔌 API 更新

### 新接口

```typescript
// V2 统一创建接口
POST /api/v1/projects/{id}/studio/create
Request: {
  type: "html" | "ppt" | "audio" | "mindmap" | "infographic" | "quiz" | "data_table",
  content: string,
  title?: string,
  options?: Record<string, unknown>,
  material_id?: number
}
Response: {
  success: boolean,
  material_id: number,
  status: "ready" | "processing" | "failed",
  result: { type, title, theme?, slide_count?, download_url? }
}

// 获取可用 Skills
GET /api/v1/studio/skills
Response: StudioSkillInfo[]
```

### 前端 API 调用

```typescript
// 新的简化调用
import { studioCreateV2, getStudioSkills } from "@/lib/studioCreateV2";

// 创建 PPT
const result = await studioCreateV2(
  projectId,
  "ppt",
  "## Q4 总结\n\n- 营收增长 25%\n- 新客户 1000+",
  "Q4 销售报告",
  { theme: "professional", slide_count: 12 }
);

// 获取 Skills 列表
const skills = await getStudioSkills();
```

## 🎨 使用示例

### 生成专业 PPT

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "type": "ppt",
    "content": "## 产品路线图\n\n### Q1 现状分析\n- 用户增长 50%\n- 营收达到预期\n\n### Q2-Q4 规划\n- 推出新功能\n- 拓展市场",
    "title": "2024 产品路线图",
    "options": {
      "theme": "professional",
      "slide_count": 15
    }
  }'
```

### 生成信息图

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -d '{
    "type": "infographic",
    "content": "人工智能发展里程碑：\n- 1956: 达特茅斯会议\n- 2012: 深度学习突破\n- 2022: ChatGPT 发布",
    "title": "AI 发展历程",
    "options": {
      "layout": "timeline",
      "theme": "modern",
      "format": "svg"
    }
  }'
```

### 生成测验

```bash
curl -X POST http://localhost:8787/api/v1/projects/123/studio/create \
  -H "Content-Type: application/json" \
  -d '{
    "type": "quiz",
    "content": "机器学习基础知识：监督学习、无监督学习、神经网络...",
    "title": "机器学习基础测验",
    "options": {
      "question_count": 15,
      "difficulty": "medium",
      "question_types": ["choice", "true_false"]
    }
  }'
```

## 🔧 前端迁移指南

### 旧代码

```typescript
import { buildStudioGenerationPrompt } from "@/lib/studioPrompts";

// 前端构建提示词
const prompt = buildStudioGenerationPrompt("slides", title);

// 调用 AI
const response = await sendStream(convId, prompt, ...);

// 调用具体接口
await chatclawApi.projects.materials.createSlidesPptx(id, {
  title,
  markdown: response,
  material_id: pendingId,
});
```

### 新代码

```typescript
import { studioCreateV2 } from "@/lib/studioCreateV2";

// 一行代码完成所有操作
const result = await studioCreateV2(
  projectId,
  "ppt",
  content,
  title,
  { theme: "professional", slide_count: 10 }
);

// 直接使用结果
if (result.success) {
  console.log(`PPT 已生成: ${result.result?.download_url}`);
}
```

## 📊 所有支持的创作类型

| 类型 | 描述 | 主要选项 |
|------|------|----------|
| **ppt** | 演示文稿 | theme, slide_count, include_notes |
| **html** | 网页 | page_type, theme, interactive |
| **audio** | 语音 | language, voice, speed |
| **mindmap** | 思维导图 | layout |
| **infographic** | 信息图 | layout, theme, format |
| **quiz** | 测验 | question_count, difficulty, question_types |
| **data_table** | 数据表 | table_type, sortable, export_formats |

## ✅ 测试验证

```bash
# 编译
go build ./...

# 测试
go test ./internal/notex/... -v

# 前端类型检查
cd web && pnpm typecheck
```

## 🚀 后续计划

1. **异步任务队列**：将 LLM 调用改为异步，支持大文件生成
2. **实时进度**：WebSocket 推送生成进度
3. **更多主题**：增加更多 PPT 主题和配色方案
4. **图片生成**：集成 AI 图片生成用于封面和插图
5. **协作功能**：多人实时编辑 Studio 创作

## 📚 参考资源

- PPT-as-code Skill: `skills/PPT-as-code/SKILL.md`
- Studio Skills: `skills/studio/*.yaml`
- API 文档: `STUDIO_ARCHITECTURE_V2.md`
