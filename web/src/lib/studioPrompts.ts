import type { Message, StudioMaterialKind } from "@/api/chatclaw";

/** Max runes of chat transcript prepended to studio user message (rest still in thread on server for non-studio turns). */
export const STUDIO_CHAT_EXCERPT_MAX_RUNES = 24000;

export function truncateToRunes(s: string, maxRunes: number): string {
  if (maxRunes <= 0) return s;
  const r = [...s];
  if (r.length <= maxRunes) return s;
  return r.slice(0, maxRunes).join("") + "\n\n... [会话节选已截断]";
}

/**
 * Build a chat excerpt for studio prompts from conversation messages (client-fetched).
 */
export function buildStudioChatExcerpt(
  messages: Message[] | null | undefined,
  opts: { maxMessages: number; summaryOnly: boolean },
): string {
  const list = Array.isArray(messages) ? messages : [];
  const relevant = list.filter((m) => m.role === "user" || m.role === "assistant");
  if (opts.summaryOnly) {
    const lastAsst = [...relevant].reverse().find((m) => m.role === "assistant");
    if (!lastAsst?.content?.trim()) return "";
    return `（会话摘要：取最近一条助手回复；需要完整对话时请关闭「仅摘要」）\n\n${lastAsst.content.trim()}`;
  }
  const n = Math.max(1, opts.maxMessages);
  const tail = relevant.slice(-n);
  return tail
    .map((m) => `${m.role === "user" ? "用户" : "助手"}: ${m.content}`)
    .join("\n\n");
}

export type StudioScopePrefixInput = {
  selectedDocuments: { id: number; name: string }[];
  chatExcerpt?: string;
  customInstruction?: string;
};

/**
 * Prefix prepended to the studio user message: selected doc hints + optional chat + custom instructions.
 */
export function buildStudioScopePrefix(opts: StudioScopePrefixInput): string {
  const blocks: string[] = [];
  if (opts.selectedDocuments.length > 0) {
    blocks.push(
      `【本次 Studio 选用的资料（后端仅注入这些文档的全文；请严格基于正文与下述会话节选输出）】\n${opts.selectedDocuments.map((d) => `- ${d.name} (document_id=${d.id})`).join("\n")}`,
    );
  }
  const chat = opts.chatExcerpt?.trim();
  if (chat) {
    blocks.push(`【当前会话上下文（与资料冲突时以资料正文为准）】\n\n${chat}`);
  }
  const extra = opts.customInstruction?.trim();
  if (extra) {
    blocks.push(`【额外说明】\n${extra}`);
  }
  if (blocks.length === 0) return "";
  return `${blocks.join("\n\n---\n\n")}\n\n---\n\n`;
}

/** Strip optional markdown code fences so backend PPTX builder receives raw Markdown. */
export function stripMarkdownFences(s: string): string {
  let t = s.trim();
  if (t.startsWith("```")) {
    t = t.replace(/^```(?:markdown|md)?\s*\n?/i, "").replace(/\n?```\s*$/i, "");
  }
  return t.trim();
}

/**
 * User message sent to the project conversation so the agent uses RAG over the bound library.
 */
export function buildStudioGenerationPrompt(kind: StudioMaterialKind, title: string): string {
  const t = title.trim() || "未命名";
  const tag = `[工作室 · ${kind}] ${t}`;
  const base =
    `${tag}\n\n系统已在上下文中注入「知识库文档正文」区块。你必须只根据该正文（及其中明确列出的材料）作答，用中文输出；不要用 Markdown 代码围栏包裹全文。若正文不足以完成要求，先说明缺什么，不要编造。`;
  switch (kind) {
    case "report":
      return `${base}\n\n请撰写 Markdown 报告《${t}》。结构建议：概述；分节要点（每节有实质信息）；小结。禁止写工具/渠道/MCP/飞书操作说明；禁止复述「检索」「知识库」等元话语。`;
    case "slides":
      return `[工作室 · slides] ${t}

你的**主交付物**是可直接使用的 **幻灯片 Markdown 大纲**（系统会据此保存并在 Studio 中预览；若运行环境具备演示文稿 skill 且你能写出真实 .pptx，可作为额外产物，但并非必需）。

请用中文输出，结构建议：
- 可选：最上方用 \`---\` 分隔的 Marp 风格 front matter（theme 等），没有也可。
- 每张幻灯片之间用单独一行的 \`---\` 分隔（无 Marp 时用 \`## 幻灯片标题\` 分节也可）。
- 每张内：标题用 \`##\`，要点用 \`- \` 列表，保持简洁。

严格要求：只根据已注入的知识库文档正文归纳，不要编造；不要用 Markdown 代码围栏包裹**全文**（局部代码块除外）；不要写工具/MCP 教程或复述检索过程。`;
    case "html":
      return `[工作室 · html] ${t}

请直接输出 **HTML**，不要输出 Markdown。禁止使用 Markdown 语法（如 # / ### 标题、**粗体**、| 表格线、--- 分隔线）；必须使用对应的 HTML 标签（如 <h1>、<strong>、<table>、<hr>）。
可为完整 <!DOCTYPE html> 文档或仅 body 内片段；可含内联 style。内容必须来自上下文中的知识库文档正文；正文不足时简短说明缺口，不要编造。不要写第三方渠道或 MCP/工具教程。不要用 markdown 代码围栏包裹全文。`;
    case "mindmap":
      return `[工作室 · mindmap] ${t}

重要提示：系统已将你勾选的所有知识库文档全文注入到当前上下文，你可以直接读取使用。
你输出的内容将直接用于 markmap 思维导图渲染，请严格遵循以下要求：
1. 第一行必须是根主题：# ${t}
2. 下面所有内容使用 Markdown 多级无序列表（以「- 」开头，子级使用2个空格缩进）
3. 层级深度至少2-3级，不要只有一级节点
4. 所有分支内容必须100%来自已注入的知识库文档正文，禁止编造
5. 不要添加任何解释性文字、思考过程或markdown代码围栏，直接输出内容即可
6. 如果文档正文内容不足，请明确说明缺少哪些类型的内容，禁止输出空内容`;
    case "infographic":
      return `${base}\n\n请用 Markdown 列出信息图模块与关键数据/文案要点（主题「${t}」），条目需可在正文中找到依据。`;
    case "data_table":
      return `${base}\n\n请从正文中提取可表格化的数据，用 Markdown 表格呈现；可多个表。若无法提取，说明缺哪些字段。`;
    case "quiz":
      return `${base}\n\n请基于正文出题：混合题型（选择、判断、简答），附答案与简短解析；10–20 题；Markdown 排版。`;
    case "audio":
      return `[工作室 · audio] ${t}

系统上下文已包含知识库文档正文。请先只输出「可直接用于语音合成（TTS）」的连续口语文本：
- 可用两位主持人对话体，每行前加「主持人1：」「主持人2：」；少用括号舞台说明。
- 不要输出 Markdown 标题、列表符号、代码块或链接；不要复述检索过程。
- 总字数建议控制在 3500 字以内以便一次合成；若正文过长请浓缩为 8–12 分钟口语量。
- 内容必须来自已注入的正文，禁止编造。`;
    default:
      return `${base}\n\n请完成与「${t}」相关的内容整理，使用 Markdown，结构清晰。`;
  }
}

/** Payload stored on project material after AI generation. */
export function buildStudioPayload(_kind: StudioMaterialKind, rawText: string): Record<string, unknown> {
  return { markdown: rawText.trim() };
}
