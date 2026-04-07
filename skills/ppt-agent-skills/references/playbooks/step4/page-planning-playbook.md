# Page Planning Playbook -- 单页策划稿

## 目标

制定一张从布局、字体、配图策略到卡片组织的 1280x720 物理画幅精细蓝图。**本阶段只写 JSON，不写 HTML。**

---

## Phase 1：理解当前页定位

从 `outline.txt` 中找到第 N 页的定义，明确：
- `page_goal`：这一页的核心论点（一句话，不含"和"字）
- `narrative_role`：叙事角色（cover/toc/section/evidence/comparison/process/close/cta）
- `proof_type`：论证方式（数据驱动/案例/对比/框架/步骤）

---

## Phase 2：资源发现与设计决策

运行 `resource_loader.py menu` 获取可用组件菜单后，**你是设计师，不是填表员**。不要查表套用，请回答以下设计提问来驱动你的选择：

1. **观众在这一页应该先看到什么？** → 决定你的视觉锚点和主次关系
2. **这一页的信息是怎么“流动”的？** → 决定空间布局和视觉动线
3. **这一页和上一页的视觉感受应该有什么不同？** → 决定节奏变化
4. **在菜单中的工具里，哪些能最好地服务上面 3 个答案？** → 决定 layout_hint、card_type、chart、resource_ref

> **重要**：菜单里的工具是你的调色盘，不是说明书。同样的数据可以用完全不同的工具和布局来表达，关键是你想让观众产生什么感受。

**填写 `resources` 字段时必须说明为什么选择该组件**（`resource_rationale` 字段）。

### 命名合同（必须区分 schema 枚举 与 资源文件 stem）

- `layout_hint` / `page_type`：写 validator 认可的值。`layout_hint` 推荐使用真实文件 stem，如 `hero-top`、`mixed-grid`、`l-shape`。
- 非 `content` 页优先通过 `page_type` 消费 `page-templates/`（如 `cover` / `toc` / `section` / `end`）。通常不需要再写 `layout_hint`；只有在要显式钉住模板正文时，才额外填写 `resources.page_template`。
- `card_type`：写 validator 认可的枚举，如 `data_highlight`、`image_hero`、`matrix_chart`。
- `chart.chart_type`：写 validator 认可的枚举，**使用下划线命名**，如 `metric_row`、`comparison_bar`、`stacked_bar`、`progress_bar`。
- `resources.*_refs` 与 `card.resource_ref.*`：推荐写 `references/` 中的真实文件 stem，如 `metric-row`、`comparison-bar`、`visual-hierarchy`；`resource_loader.py` 会自动做下划线/连字符归一化。
- `process` 是 schema 原生 `card_type`，但当前没有 `blocks/process.md`。若使用它，必须同时给出更强的 `layout_refs`、`principle_refs`、`director_command` 和必要的 `chart_refs` / `resource_ref`，不要假设会有专属 block 正文可加载。

### principle_refs 指导（重要：设计原则文件按场景选用）

`resources.principle_refs[]` 字段决定 HTML 阶段能否收到设计原则正文。按以下规则填写：

| 本页特征 | 应引用 |
|---------|--------|
| 数据图表主导页 | `data-visualization` |
| 多卡片排版，需要层次感 | `visual-hierarchy` |
| 封面/章节页，需要情绪校准 | `color-psychology` |
| 信息超密、担心认知负担 | `cognitive-load` |
| 叙事转折页（从问题到方案）| `narrative-arc` |
| 任何页面的排版构图优化 | `composition` |
| 不确定选哪个 | `design-principles-cheatsheet`（综合速查）|

在 planning JSON 中写法示例：
```json
"resources": {
  "principle_refs": ["visual-hierarchy", "composition"],
  "layout_refs": ["hero-top"],
  "block_refs": [],
  "chart_refs": ["kpi"]
}
```

填写后，`resource_loader.py resolve` 会自动把对应原则文件的完整正文注入 HTML 阶段的上下文。

---

## Phase 3：`planningN.json` 结构合同（强制）

你的输出必须是**可直接被 `planning_validator.py` 校验的 JSON**。以下是 schema 骨架（**只展示结构，不展示设计决策** -- 布局、卡片类型、视觉风格全部由你自主决定）：

```json
{
  "page": {
    "slide_number": "<页码>",
    "page_type": "<cover/toc/section/content/end>",
    "narrative_role": "<叙事角色>",
    "title": "<页标题>",
    "page_goal": "<一句话核心论点>",
    "audience_takeaway": "<观众带走什么>",
    "visual_weight": "<1-10 信息密度>",
    "layout_hint": "<你的布局选择>",
    "layout_variation_note": "<与上一页的差异点，自由发挥>",
    "focus_zone": "<视觉焦点区域描述>",
    "negative_space_target": "<high/medium/low>",
    "page_text_strategy": "<文字策略>",
    "rhythm_action": "<推进/爆发/缓冲/收束>",
    "must_avoid": ["<你认为这页最危险的平庸设计陷阱>"],
    "variation_guardrails": {
      "same_gene_as_deck": "<哪些元素跨页保持统一>",
      "different_from_previous": ["<与上一页的具体变化维度>"]
    },
    "director_command": {
      "mood": "<你为这页设定的情绪基调>",
      "spatial_strategy": "<你的空间编排策略>",
      "anchor_treatment": "<你怎么处理视觉锚点>",
      "techniques": ["<你选用的技法编号>"],
      "prose": "<用电影镜头语言描述这页的视觉感受>"
    },
    "decoration_hints": {
      "background": {"feel": "<>", "restraint": "<>", "techniques": ["<>"]},
      "floating": {"feel": "<>", "restraint": "<>", "techniques": ["<>"]},
      "page_accent": {"feel": "<>", "restraint": "<>", "techniques": ["<>"]}
    },
    "source_guidance": {
      "brief_sections": ["<素材引用路径>"],
      "citation_expectation": "<引用策略>",
      "strictness": "<证据边界>"
    },
    "resources": {
      "page_template": "<null 或页面模板 ref>",
      "layout_refs": ["<你的 layout ref>"],
      "block_refs": [],
      "chart_refs": ["<你选用的 chart ref>"],
      "principle_refs": ["<你需要的设计原则>"],
      "resource_rationale": "<为什么选这些资源，必须说明理由>"
    },
    "cards": [
      {
        "card_id": "<s页码-role-序号>",
        "role": "<anchor/support/context>",
        "card_type": "<你的卡片类型选择>",
        "card_style": "<你的视觉变体选择>",
        "argument_role": "<claim/evidence/context>",
        "headline": "<精炼标题>",
        "body": ["<正文字符串数组>"],
        "data_points": [{"label": "<>", "value": "<>", "unit": "<>", "source": "<>"}],
        "chart": {"chart_type": "<你的图表类型>"},
        "content_budget": {"headline_max_chars": 12, "body_max_bullets": 3, "body_max_lines": 5},
        "image": {
          "mode": "<generate/provided/manual_slot/decorate>",
          "needed": "<true/false>",
          "usage": "<null 或图片用途>",
          "placement": "<null 或放置位置>",
          "content_description": "<null 或描述>",
          "source_hint": "<null 或路径>",
          "decorate_brief": "<装饰说明>"
        },
        "resource_ref": {"chart": "<>", "principle": "<>"}
      }
    ],
    "workflow_metadata": {
      "stage": "planning",
      "workflow_version": "2026.03.31-v4",
      "planning_schema_version": "4.0",
      "planning_packet_version": "4.0",
      "planning_continuity_version": "4.0"
    }
  }
}
```

> **重要提醒**：以上每个 `<>` 占位符都需要你根据本页的内容、受众、风格、叙事节奏做出自主的设计决策。你是设计师，不是填写模板的文员。

### 必填字段与枚举底线

- 顶层页字段至少要有：`slide_number`、`page_type`、`title`、`page_goal`、`cards`、`visual_weight`、`director_command`、`decoration_hints`、`resources`、`workflow_metadata`。
- `page_type`：`cover` / `toc` / `section` / `content` / `end`
- `narrative_role`：与 outline 的叙事角色对齐，使用 `cover` / `toc` / `section` / `evidence` / `comparison` / `process` / `close` / `cta`
- 内容页必须有 `layout_hint`，并从 validator 认可的集合中选，如 `single-focus`、`symmetric`、`asymmetric`、`three-column`、`primary-secondary`、`hero-top`、`mixed-grid`、`l-shape`、`t-shape`、`waterfall`
- `cards[].role`：`anchor` / `support` / `context`
- `cards[].card_style`：`accent` / `elevated` / `filled` / `outline` / `glass` / `transparent`
- `cards[].body` 必须是**字符串数组**，不要写成单个字符串
- `cards[].data_points` 必须是对象数组；有数字时尽量带 `source`
- `cards[].content_budget` 必须是对象；哪怕是最小对象也要显式写出
- `cards[].image.needed = true` 时，`usage` / `placement` / `content_description` / `source_hint` 都必须填写；否则这些字段应为 `null`

---

## Phase 4：图片策略决策（必须明确，不得含糊）

| 模式 | 适用场景 | 必填字段 |
|------|---------|---------|
| `generate` | 封面页、章节页、需要强视觉冲击的核心页 | `image.needed=true`、`usage`、`placement`、`content_description`、`source_hint`（目标落盘路径）、`image.prompt`（英文图生图提示词） |
| `provided` | 用户已提供图片/品牌图库/截图 | `image.needed=true`、`source_hint`（真实本地路径）|
| `manual_slot` | 用户后续自己补图，先占位 | `image.needed=false`、`image.slot_note` 说明槽位位置、比例、替换建议 |
| `decorate` | 数据页、逻辑页、纯排版页 | `image.needed=false`、`image.decorate_brief` 说明内部视觉语言（SVG/渐变/色块/水印/字体装饰）|

**禁止留模棱两可的 mode。选定后不得在 HTML 阶段临时改变。**

---

## Phase 5：你是设计师，不是填表员

> **核心理念**：上面的 Phase 2 菜单和 Phase 3 schema 是你的工具箱和表格结构，不是你的设计决策。你的工作不是"查表填空"，而是"为这一页创造一个独一无二的视觉方案"。

**你的创意自由度：**
- `layout_hint` 不是铁律，它只是重力场方向的提示。HTML 阶段可以完全重构它
- `card_type` 和 `chart_type` 决定你用什么工具，但怎么用这个工具完全由你决定
- `director_command` 是你的电影镜头笔记 -- 写得越具体越有画面感，HTML 阶段越能还原你的视觉构想
- `must_avoid` 是你对平庸的主动拒绝 -- 每页至少写 1 条真正有意义的禁区

**后续保障**：你在此阶段的所有创意决策都有像素级图审兜底，不必担心冲破边界。

---

## Phase 6：cards 字段填充规范

每张卡片必须包含：
- `card_id`：稳定唯一，建议 `s{页码}-{anchor|support|context}-{序号}`
- `role`：`anchor` / `support` / `context`
- `card_type`：validator 枚举值，如 `text` / `data` / `list` / `process` / `data_highlight` / `timeline` / `diagram` / `quote` / `comparison` / `people` / `image_hero` / `matrix_chart`
- `card_style`：6 种合法视觉变体之一
- `headline`：标题（精炼，不超过 12 字）
- `body`：正文字符串数组，不能为空
- `data_points`：如有数值则填对象数组
- `content_budget`：内容预算对象
- `image`：完整图片合同对象，带 `mode`
- `resource_ref`：需要定向绑定某个 block/chart/principle 时写这里
- `image.slot_note` / `image.decorate_brief` / `image.prompt`：按图片模式按需补充

可选但推荐：
- `argument_role`
- `chart`

**不得出现空 `body` 的卡片。**

---

## Phase 7：设计意图传递字段

不要被僵化的模板束缚。请使用以下字段向 HTML 阶段传递你的核心创意和自由发挥意图，它们是后续创意落地的灵感基石：

- `focus_zone`：提议的主张和视觉焦点区域
- `must_avoid`：明确提配 HTML 阶段不要陷入的平庸模板化设计
- `director_command`：给出富有创意性的结构、锚点和高级技法方向
- `decoration_hints`：描述装饰强度与视觉层次
- `source_guidance`：约束证据边界与引用期望
- `resources` / `resource_ref`：推荐消费的组件资源

---

## Phase 8：自审（强制）

运行 `planning_validator.py`，直到零 ERROR：

```bash
python3 SKILL_DIR/scripts/planning_validator.py $(dirname PLANNING_OUTPUT) --refs REFS_DIR --page PAGE_NUM
```

- ERROR 必须全部修复才能 FINALIZE
- WARNING 建议修复，不强制
- 自审通过后立即发送 FINALIZE，然后等待 HTML 阶段指令
