# Studio Skills 集成指南

## 概述

我们已经将根目录下的三个 YAML skill 文件集成到项目中：

- `studio-audio.yaml` → `skills/studio/studio-audio.yaml` (TTS/音频生成)
- `studio-html.yaml` → `skills/studio/studio-html.yaml` (HTML页面生成)
- `studio-ppt.yaml` → `skills/studio/studio-ppt.yaml` (PPT演示文稿生成)

## 文件变更

### 1. 新增文件

- `internal/notex/studio_skill_integration.go` - Studio Skill 集成模块
  - 提供 YAML skill 解析
  - 统一生成请求处理
  - 关键词匹配
  - API 端点

### 2. 修改文件

- `internal/notex/skills_domain.go`
  - 扩展 skill 扫描支持 `.yaml` 文件
  - 新增 `parseSkillYAML()` 函数

- `internal/notex/server_core.go`
  - 注册新的 API 端点:
    - `GET /api/v1/studio/skills` - 列出可用的 studio skills
    - `POST /api/v1/studio/generate` - 统一的生成接口

## API 使用

### 列出 Studio Skills

```bash
GET /api/v1/studio/skills

Response:
[
  {
    "type": "audio",
    "name": "studio-audio",
    "version": "1.0",
    "description": "将文本转换为高质量语音音频...",
    "trigger_keywords": ["生成音频", "文字转语音", "TTS"],
    "input_fields": ["text", "language", "voice", "speed"],
    "output_fields": ["job_id", "status", "audio_url"]
  }
]
```

### 统一生成接口

```bash
POST /api/v1/studio/generate

Request:
{
  "type": "html",
  "content": "要转换的内容",
  "project_id": 123,
  "title": "我的网页",
  "options": {
    "theme": "dark",
    "page_type": "article"
  }
}

Response:
{
  "status": "pending",
  "result": {
    "skill_name": "studio-html",
    "content_type": "html"
  }
}
```

## 现有接口兼容性

现有的生成接口保持不变，完全兼容：

- `POST /api/v1/projects/{id}/materials/slides-pptx` - PPT生成
- `POST /api/v1/projects/{id}/materials/studio-html` - HTML生成
- `POST /api/v1/projects/{id}/materials/studio-audio` - 音频生成
- `POST /api/v1/projects/{id}/materials/studio-mindmap` - 思维导图生成

## Skill 定义结构

每个 YAML skill 文件包含：

```yaml
name: studio-<type>
version: "1.0"
description: "描述"
trigger_keywords:
  - 关键词1
  - 关键词2
tools:
  - tool_name
input_schema:
  required: [field1]
  optional: [field2]
output_schema:
  fields: [result_field]
---
# Prompt 模板和详细说明
```

## 后续扩展

要添加新的 studio 生成类型：

1. 创建新的 YAML 文件 `skills/studio/studio-<type>.yaml`
2. 按照现有格式定义 schema 和 prompt
3. 在 `studio_skill_integration.go` 中添加对应的处理逻辑
4. 重启服务即可自动加载

## 技术细节

- Skill 文件在启动时扫描并缓存
- 支持热刷新（通过 `POST /api/v1/skills/refresh`）
- 关键词匹配支持中文和英文
- 输入验证基于 YAML 中定义的 schema
