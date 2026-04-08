# 采访问卷共享核心

> 本文件是 Step 0 的共享采访内容合同，不直接作为运行时 prompt 发给主 agent。
> 运行时应按能力选择：
> - `tpl-interview-structured-ui.md`
> - `tpl-interview-text-fallback.md`

# 采访问卷共享核心

> 本文件是 Step 0 的共享采访内容合同，不直接作为运行时 prompt 发给主 agent。
> 运行时应按能力选择 `tpl-interview-structured-ui.md` 或 `tpl-interview-text-fallback.md`。

## 核心采访目标（执行指南）

作为系统首个守门节点，必须以最高效的轮次获取极高信噪比的输入。核心目标：不与用户寒暄，直接锁定能左右大纲结构、视觉风格和管线（Pipeline）分支的关键维度参数。

## 必须覆盖的核心维度域（四组）

你向用户抛出的选项，必须精准涵盖以下四个维度域：

### A. 业务场景与传达目标
*左右内容深度与叙事基调*
- `scenario` (场景): 新人介绍 / 内部汇报 / 社区宣讲 / 招商合作 / 融资路演 / 大众科普等
- `audience` (身份与受众视角): “你是谁，要在上面向谁讲？” (如：一线操盘手向高层要资源 / 业务一号位向专家画大饼 / 讲师向小白泛大众布道)
- `target_action` (用户心智转化动作): 建立认知 / 促成意向 / 愿意加入 / 纯信息同步

### B. 结构密度与生产管线
*左右大纲页数、图文排布与数据源获取*
- `expected_pages` (期望页数): 5-10页 / 10-20页 / 20-30页宽幅 / 自由发挥
- `page_density` (版面信息密度): 少而精 / 适中 / 容量极大
- `material_strategy` (数据源头分支): [Research(全网扩写)] 或 [非Research(仅限当前提供资料)]
- `must_include` / `must_avoid`: (可要求用户补充听众散场时必须记住的【唯一核心主张 The One Big Idea】，以及不可触碰的红线禁忌)

### C. 视觉审美与资产策略
*左右后续 Style/HTML 生成器的美学锁*
- `visual_style` (整体风格): 极简商务 / 科技极客 / 轻量活泼 / 自动匹配
- `language_mode` (落地产物语言): 中文 / 英文 / 中英混排
- `imagery_strategy` (配图资产策略): decorate(纯装饰) / generate(AI配文生成) / provided(自带资产) / manual_slot(占位预留)
- `brand_constraints`: (品牌视觉禁忌限制)

### D. 构建环境与工程卡口
- `success_criteria` (用户评价标准)
- `subagent_model_strategy` (子系统模型接力): 继承主代理 / 降级指定等

## 落点契约要求

所有问卷结果必须映射到以下两份产物，作为后续子代理的真源输入：

1. `interview-qa.txt`
   保留用户原意。为通过 `contract_validator.py` 强校验，结尾必须附加以下英文锚点块（缺一不可）：
   `scenario`, `audience`, `target_action`, `expected_pages`, `page_density`, `style`, `brand`, `must_include`, `must_avoid`, `language`, `imagery`, `material_strategy`
   
2. `requirements-interview.txt`
   脱水的纯净参数组，同样必须包含上方的这 12 个锚点维度，并带上你收集到的丰富化选项值，以此指引各层级 Pipeline 的排版和选型走向。
