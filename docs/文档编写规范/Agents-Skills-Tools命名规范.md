# YouMind Agent / Skill / Tool 命名与目录规范

> 本规范用于 YouMind 项目中 **Agent / Skill / Tool 的统一命名、目录组织与标识规则**。
> Skill 命名参考 [https://agentskills.io/specification](https://agentskills.io/specification)，结合 YouMind 实际工程结构制定。
>
> **项目技术栈**：Go 1.25 + Eino (Agent 框架) + React 19

---

## 一、总体命名原则（全局适用）

1. **机器优先，可被 LLM 稳定理解与生成**
2. **名称即能力语义，不依赖上下文补全**
3. **避免缩写、避免歧义、避免多义词**
4. **命名必须稳定，可作为长期 API / 协议的一部分**
5. **人类可读，但不追求自然语言完整句**

---

## 二、Skill 命名规范（严格遵循 Agent Skills 标准）

### 2.1 Skill 的定位

Skill = **可被 Agent 在“思考阶段”引用的能力说明文档**

* 不直接执行代码
* 不保存状态
* 不做流程调度
* 描述 *“如何做一类事”*

Skill 的本质是：

> **给 LLM 的可复用操作说明书（Instructional Capability）**

---

### 2.2 Skill 名称（name 字段 & 目录名）【强制】

Skill 的 `name` 字段 **必须同时作为 Skill 目录名**，并满足以下规则：

**允许字符**

* 小写字母：`a-z`
* 数字：`0-9`
* 连字符：`-`

**强制规则**

* 只能使用 `a-z0-9-`
* 不允许大写字母
* 不允许下划线 `_`
* 不允许空格
* 不允许以 `-` 开头或结尾
* 不允许连续 `--`
* 长度建议：`3–64` 字符

**推荐语义结构**

```
<domain>-<verb>-<object>
<verb>-<object>
```

**YouMind Skill 示例**

* `deep-research`
* `image-generation`
* `video-generation`
* `podcast-generation`
* `ppt-as-code`
* `chart-visualization`
* `company-research`
* `paper-research`
* `academic-paper-review`
* `code-documentation`
* `web-design-guidelines`
* `skill-creator`

**错误示例**

* `DeepResearch`      ❌（大写）
* `image_generation`  ❌（下划线）
* `generateImage`     ❌（驼峰）
* `deep--research`    ❌（连续连字符）

---

### 2.3 Skill 文件结构（强制）

```
skills/
├── public/                    # 公开 Skills
│   ├── deep-research/
│   │   └── SKILL.md
│   ├── image-generation/
│   │   └── SKILL.md
│   ├── ppt-as-code/
│   │   └── SKILL.md
│   └── ...
└── <other-groups>/            # 其他 Skill 分组
    └── <skill-name>/
        └── SKILL.md
```

* 目录名 = `name`
* 文件名固定为 `SKILL.md`
* 一个 Skill = 一个目录
* YouMind Skills 按功能分组存放（如 public/）

---

### 2.4 SKILL.md Frontmatter 命名要求

```yaml
---
name: section-content-write
description: Write detailed documentation content for a specific section based on explored code context.
license: MIT
allowed-tools: read_file search_code
metadata:
  version: "0.1"
  author: openDeepWiki
---
```

| 字段            | 说明                          |
| --------------- | ----------------------------- |
| `name`          | 必须与目录名完全一致          |
| `description`   | 清晰描述“什么时候用 + 做什么” |
| `allowed-tools` | 明确声明允许使用的 Tool 能力  |

---

## 三、Tool 命名规范（执行型能力）

### 3.1 Tool 的定位

Tool = **可被 LLM 直接调用的、确定性执行能力**

* 有明确输入 / 输出（基于 Eino 的 Tool 接口）
* 由 Go 代码实现
* 无自然语言歧义
* 位于 `pkg/tools/` 目录下

---

### 3.2 Tool 命名规则

**格式**

```
<verb>_<object>
```

**命名规则**

* 全小写
* 使用下划线 `_`
* 动词开头
* 明确操作对象

**YouMind Tool 示例**

内置工具（`pkg/tools/builtin/`）：
* `bash` - 执行 Bash 命令
* `file` - 文件读写操作
* `web` - Web 请求与搜索
* `tavily` - Tavily 搜索 API
* `view_image` - 图像查看与分析

工具注册与管理（`pkg/tools/`）：
* `registry` - 工具注册表
* `present` - 工具呈现
* `path_access` - 路径访问控制
* `tool_search` - 工具搜索

---

## 四、Agent 命名规范（角色型能力）

### 4.1 Agent 的定位

Agent = **具备角色目标的长期执行单元**

* 有 system prompt
* 基于 Eino Agent 框架实现
* 绑定一组 Skills
* 允许使用一组 Tools
* 可被调度、并行、恢复
* 位于 `pkg/agent/` 目录下

---

### 4.2 Agent 命名规则

**格式**

```
<domain>-agent
<role>-agent
```

**YouMind Agent 组件**

位于 `pkg/agent/`：
* `react` - ReAct 循环实现
* `subagent` - 子 Agent 管理
* `types` - Agent 类型定义与配置
* `view_image` - 图像理解 Agent

YouMind 采用 **Eino** 作为 Agent 框架，Agent 通过组合 Skills 和 Tools 实现特定功能。

典型 Agent 应用场景：
* `workflow-agent` - 执行工作流的 Agent
* `research-agent` - 研究分析 Agent
* `generation-agent` - 内容生成 Agent
* `review-agent` - 审查 Agent

---


## 五、YouMind 命名清单参考

### Skills（位于 skills/ 目录）

内容生成类：
* `deep-research`
* `image-generation`
* `video-generation`
* `podcast-generation`
* `newsletter-generation`

PPT/文档类：
* `ppt-as-code`
* `report-to-ppt`
* `chart-visualization`

研究分析类：
* `company-research`
* `paper-research`
* `academic-paper-review`
* `github-deep-research`

开发工具类：
* `code-documentation`
* `frontend-design`
* `web-design-guidelines`
* `skill-creator`
* `find-skills`

### Tools（位于 pkg/tools/）

内置工具（`pkg/tools/builtin/`）：
* `bash`
* `file`
* `web`
* `tavily`
* `view_image`

工具管理：
* `registry`
* `present`
* `path_access`
* `tool_search`
* `skills`
* `subagent`

### Agent 组件（位于 pkg/agent/）

核心实现：
* `react` - ReAct 推理循环
* `subagent` - 子 Agent 调用
* `types` - 类型定义与配置
* `view_image` - 图像理解

应用层 Agent（工作流中）：
* `workflow-agent`
* `research-agent`
* `generation-agent`
* `data-analysis-agent`

---

## 六、强制执行建议

* 校验 Skill 名是否符合 agentskills.io 规则
* 禁止 Agent / Tool 与 Skill 混用命名风格

---

> 本规范用于 **指导 AI 在 YouMind 项目中自动生成代码 / Skill / Agent 定义**，所有约束均为工程级强约束。
>
> **参考目录**：
> - Skills: `skills/`
> - Tools: `pkg/tools/`
> - Agent 框架: `pkg/agent/`
