# PPT as Code

中文说明。英文版请看 [README.md](./README.md)。

`PPT as Code` 是一个面向内容创作者和产品表达场景的 HTML 演示文稿 skill。

它不是把“网页”和“PPT”硬拼在一起，而是把一套真正适合演示的工作流结构化出来：

- 先锁主题和结构
- 再锁风格和脚本
- 再处理图片和页面节奏
- 最后再落成 HTML

## 这个 Skill 的优点

- 更像做一套演示，而不是拼一篇网页长文。
- 不会一上来就乱写代码，先把关键决策锁住。
- 轻模式也有视觉方向，不会只给你一个技术骨架。
- 搜图是按每一页的核心判断来做，不是按整套 deck 大主题瞎搜。
- 没有网络、不能下载图片、不能写文件时，也不会直接卡死。
- `advanced` 先出静态版，再决定要不要补动态，节奏更稳。

## 核心特性

### 1. 三档工作流

- `quick`
  适合最小可用版本、快速原型、先跑起来再升级。
- `basic`
  适合先确认主题拆解，再确认脚本，再确认图片方案，最后再落 HTML。
- `advanced`
  适合更强的视觉锁定、更完整的参考图流程、静态优先、后续可补动态。

### 2. 默认对话优先

开源版默认不会假设你的仓库里一定能写文件。

它会先在对话中产出这些阶段性内容：

- brief
- 主题拆解
- 风格选项
- 脚本
- 图片方案
- HTML 路线或静态结果

只有在下面两种情况下，才建议把这些内容真正写进仓库：

- 用户明确要求落地成文件
- 当前仓库结构明显适合这种文件化工作流

### 3. 有网络就增强，没网络也能继续

如果环境支持浏览：

- `advanced` 可以先找 3 个真实 PPT / slide design 参考图
- 可以按页搜索图片
- 可以做更完整的参考图驱动流程

如果环境不支持浏览：

- 不会卡在“必须先搜到参考图”
- 会直接根据风格词、用户给的灵感和主题，生成 structured design constraints
- 会给搜索词和图片意图，而不是假装已经搜过图

### 4. 图片逻辑是按页走，不是按主题走

这个 skill 不会拿整套 deck 的大主题直接搜图。

它会先：

1. 压出这一页到底想说什么
2. 提炼这页自己的关键词
3. 再拿关键词加风格方向去搜

关键词规则：

- `basic`：每页 1 到 2 个关键词
- `advanced`：每页 3 到 4 个关键词

### 5. 动态是第二阶段，不是默认第一阶段

在 `advanced` 里，动态效果不是默认先做。

顺序是：

1. 先锁风格
2. 先锁脚本和图片
3. 先出静态 HTML
4. 用户确认后，再决定要不要补动态

这样能避免很多“动画先飞起来了，但内容和结构其实还没锁”的问题。

## 原理很简单

这个 skill 的底层原理，其实就几条：

- 不要过早锁定最终页面
- 不要把“网页排版”误当成“演示设计”
- 不要用整套 deck 的大主题去粗暴搜图
- 不要把高风险决策混成一步做完
- 不要把网络能力、下载能力、写文件能力当成理所当然

所以它的流程设计是：

1. 先补缺口
2. 再做阶段性产物
3. 再确认高风险决策
4. 最后才进入 HTML

如果你把它理解成一句话，就是：

先把演示逻辑做对，再把页面做出来。

## 它适合什么场景

- 想用 HTML 做 stage-like 演示的人
- 想把 deck workflow 结构化的人
- 有 rough notes，但没有清晰拆解和风格方向的人
- 想先锁静态版，再决定要不要补动画的人
- 想把图片逻辑做得更贴页，而不是更花哨的人

## Package 结构

```text
ppt-as-code-open/
|-- SKILL.md
|-- README.md
|-- README-zh.md
|-- LICENSE
|-- CONTRIBUTING.md
|-- agents/
|   `-- openai.yaml
|-- references/
|   |-- quick-mode.md
|   |-- basic-mode.md
|   |-- advanced-mode.md
|   |-- visual-and-images.md
|   |-- component-libraries.md
|   `-- evolution-log.md
`-- workflows/
    |-- mode-delivery.md
    `-- evolution-writeback.md
```

## 三种模式分别会给什么

### Quick

一般会给你：

- 轻量 brief
- 3 到 4 个风格方向
- 一个推荐方向
- 最小 slide 结构
- 一个最小 HTML 路线或 prompt pack

### Basic

一般会给你：

- brief
- 主题拆解
- 风格选项
- 确认过的脚本
- 图片方案
- 静态 HTML

### Advanced

一般会给你：

- brief
- 风格选项
- 参考图分支或无网络 fallback
- structured design constraints
- 确认过的脚本
- 图片方案
- 静态 HTML
- 可选的动态补全

## 文件落地策略

开源版是保守的。

它默认不假设自己可以直接写你的仓库。

默认行为是：

- 先在对话里输出阶段性内容
- 只有用户明确要求、或者仓库结构明显支持时，才真正写出这些文件

可能写出的文件包括：

- `deck_brief.md`
- `theme_breakdown.md`
- `style_options.md`
- `deck_script.md`
- `image_plan.md`
- `index.html`
- `assets/`

## 网络与下载策略

### 能浏览时

它可以：

- 在 `advanced` 里找 3 个参考图
- 按页搜图
- 用真实参考图锁视觉方向

### 不能浏览时

它会：

- 跳过 web reference 分支
- 直接根据风格词和用户灵感生成 structured design constraints
- 给出搜索词、图片意图和实现约束

### 不能下载时

它会：

- 保留图片链接或搜索字符串
- 标记为需要手动下载
- 不让整个流程卡住

## 示例 Prompt

### Quick

```text
Use ppt-as-code to help me build a fast HTML deck about AI workflow design for product teams.
I want something lightweight and stage-like, not a long article.
Start with a quick mode route.
```

### Basic

```text
Use ppt-as-code to help me build a presentation about why AI-native teams need new operating habits.
I have rough notes but no clear structure or style yet.
Please use a basic workflow and confirm the breakdown before writing HTML.
```

### Advanced

```text
Use ppt-as-code to build a premium HTML deck about AI product differentiation.
I want stronger design direction and a static-first workflow.
If browsing is available, give me real presentation references.
If not, synthesize the design constraints directly from the style direction.
```

## 维护与贡献

这份开源版把“运行时文档”和“维护者文档”分开了。

如果你想看维护规则，可以继续看：

- [README.md](./README.md)
- [CONTRIBUTING.md](./CONTRIBUTING.md)
- [workflows/evolution-writeback.md](./workflows/evolution-writeback.md)
- [references/evolution-log.md](./references/evolution-log.md)

维护原则很简单：

稳定规则，直接写回主文档。
不要长期堆在 log 里。
