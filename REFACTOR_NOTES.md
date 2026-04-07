# YouMind / Minote 源码分析与重构建议

## 一、整体评估

基于 README、CLAUDE.md、AGENTS.md 及项目目录结构的全面分析，项目整体架构清晰、技术选型现代，
但在以下几个维度存在可优化空间。

---

## 二、架构层面问题与建议

### 2.1 Studio 模块（最高优先级）

**现状问题**
- audio / html / ppt 三种生成能力耦合在同一 handler 或 skill 中
- 缺少统一的任务状态机（无 pending → processing → done/failed 流转）
- 无明确的 input/output schema 约束，前后端对接靠约定
- TTS 有两套服务（tts / volc-tts），调用方式不统一
- 错误信息不结构化，前端难以展示有意义的错误

**重构方案**
```
skills/
├── studio-audio.yaml   ← 新增（本次交付）
├── studio-html.yaml    ← 新增（本次交付）  
└── studio-ppt.yaml     ← 新增（本次交付）

pkg/studio/
├── types.go            ← StudioJob, StudioType 等共享类型
├── audio.go            ← TTS 调用封装（统一 tts/volc-tts 接口）
├── html.go             ← HTML 生成封装
└── ppt.go              ← PPT 生成封装（python-pptx gRPC 或子进程）

internal/notex/
├── studio_audio_handler.go
├── studio_html_handler.go
└── studio_ppt_handler.go

API 设计：
POST /api/v1/studio/audio/generate
POST /api/v1/studio/html/generate
POST /api/v1/studio/ppt/generate
GET  /api/v1/studio/jobs/:id
GET  /api/v1/studio/jobs/:id/stream  (SSE)
```

### 2.2 服务间通信

**现状问题**
- 多个 cmd/ 服务之间通信方式不明确（HTTP？gRPC？直接调用？）
- 存在直接函数调用和 HTTP 调用混用的风险

**建议**
- 内部服务通信统一使用 gRPC（已有 protobuf 依赖）
- 对外暴露统一通过 Gateway 路由，服务间不直接 HTTP 调用
- 定义清晰的 proto 文件放在 `proto/` 目录

### 2.3 Skills 系统

**现状问题**
- skill YAML 格式缺少 input_schema / output_schema 约束
- trigger_keywords 匹配规则不透明
- 无版本管理（升级 skill 无法灰度）

**建议**
- 统一 YAML schema（已在新 skill 文件中实现）
- 添加 `version` 字段，支持同名 skill 多版本共存
- trigger_keywords 支持正则或 embedding 语义匹配

---

## 三、代码质量问题与建议

### 3.1 错误处理不一致

**现状**：不同 handler 的错误格式和状态码不统一

**建议**：
```go
// pkg/apierror/error.go — 统一错误定义
var (
    ErrNotFound        = &APIError{Code: "NOT_FOUND", Status: 404}
    ErrUnauthorized    = &APIError{Code: "UNAUTHORIZED", Status: 401}
    ErrInvalidInput    = &APIError{Code: "INVALID_INPUT", Status: 400}
    ErrStudioTimeout   = &APIError{Code: "STUDIO_TIMEOUT", Status: 504}
)
```

### 3.2 配置管理

**现状**：环境变量和 config.yaml 混用，优先级文档未落地到代码

**建议**：
- 使用 `github.com/spf13/viper` 统一管理（支持 env、yaml、默认值）
- 为每个服务创建独立的配置结构体（避免全局 config 包）

### 3.3 日志规范

**现状**：`log.Printf` 和 `klog` 混用

**建议**：统一使用结构化日志（`slog` 标准库，Go 1.21+）：
```go
slog.Info("studio generate started",
    "type", req.Type,
    "job_id", jobID,
    "user_id", userID,
)
```

### 3.4 测试覆盖

**现状**：缺少针对 Studio 模块的集成测试

**建议**：
```go
// pkg/studio/audio_test.go
func TestAudioGenerate(t *testing.T) {
    // 使用 mock TTS 服务
    svc := NewAudioService(mockTTS)
    job, err := svc.Generate(ctx, AudioInput{Text: "hello", Voice: "zh-female-1"})
    assert.NoError(t, err)
    assert.Equal(t, StatusDone, job.Status)
}
```

---

## 四、前端优化建议

### 4.1 Studio 组件拆分

**现状**：Studio 相关组件可能耦合

**建议**：
```
web/src/components/studio/
├── AudioStudio.tsx      ← 音频生成界面
│   ├── VoiceSelector.tsx
│   ├── SpeedControl.tsx
│   └── AudioPlayer.tsx
├── HtmlStudio.tsx       ← HTML 生成界面
│   ├── PageTypeSelector.tsx
│   ├── ThemeSelector.tsx
│   └── HtmlPreview.tsx  (iframe)
└── PptStudio.tsx        ← PPT 生成界面
    ├── OutlineEditor.tsx
    ├── ThemeSelector.tsx
    └── SlideCount.tsx
```

### 4.2 任务状态管理

```tsx
// hooks/useStudioJob.ts
function useStudioJob(jobId: string) {
  return useQuery({
    queryKey: ['studio-job', jobId],
    queryFn: () => fetchStudioJob(jobId),
    refetchInterval: (data) => 
      data?.status === 'processing' ? 2000 : false,
  });
}
```

---

## 五、安全加固建议

1. **Studio 输入大小限制**：服务端强制校验（文本 5000 字、文件 20MB），不依赖前端
2. **文件 URL 访问控制**：生成的 audio/html/ppt 文件应带签名 URL（MinIO presigned URL），避免未授权访问
3. **LLM Prompt Injection**：HTML 生成时对用户输入做 HTML 实体转义，防止生成恶意代码
4. **速率限制**：每用户 Studio 任务并发数 ≤ 3，防止滥用

---

## 六、实施优先级

| 优先级 | 任务 | 工作量 |
|--------|------|--------|
| P0 | Studio skill 拆分（本次已交付 YAML） | 已完成 |
| P0 | `pkg/studio/` 公共包实现 | 1-2天 |
| P0 | Studio 独立 handler + 统一任务状态机 | 2-3天 |
| P1 | 统一错误处理 (`pkg/apierror`) | 0.5天 |
| P1 | 前端 Studio 组件按类型拆分 | 1-2天 |
| P1 | 统一日志（slog） | 0.5天 |
| P2 | gRPC 内部通信统一 | 3-5天 |
| P2 | Studio 集成测试 | 1-2天 |
| P3 | Skill YAML 版本管理 | 1天 |

---

## 七、交付物清单

本次交付：
```
CLAUDE.md                  ← 重构后的 AI 协作规范
AGENTS.md                  ← 重构后的架构参考文档
skills/studio-audio.yaml   ← 音频生成独立 skill
skills/studio-html.yaml    ← HTML 生成独立 skill
skills/studio-ppt.yaml     ← PPT 生成独立 skill
REFACTOR_NOTES.md          ← 本文档
```
