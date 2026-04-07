/**
 * Remove model "chain-of-thought" and retrieval boilerplate from slide outline markdown
 * before PPTX generation and preview. Keeps ## / - structure when possible.
 */

const FLUFF_SUBSTRINGS = [
  "我需要先检索知识库",
  "我需要检索知识库",
  "我我需要",
  "您您提到的",
  "进一步确认",
  "主题范围较广",
  "了解当前关联了哪些资料",
  "知识库检索结果",
  "根据知识库检索",
  "根据上述检索",
  "基于检索结果",
  "未找到相关资料",
  "先检索知识库",
  "检索知识库，了解",
];

function isFluffLine(line: string): boolean {
  const t = line.trim();
  if (!t) return true;
  for (const k of FLUFF_SUBSTRINGS) {
    if (t.includes(k)) return true;
  }
  if (t.includes("我需要先检索") && t.includes("知识库")) return true;
  return false;
}

function stripBoldMarkers(s: string): string {
  return s.replace(/\*\*([^*]+)\*\*/g, "$1");
}

/** Dedupe consecutive identical lines (model loops), keep headings. */
function dedupeConsecutiveLines(text: string): string {
  const lines = text.split(/\r?\n/);
  const out: string[] = [];
  let prev = "";
  for (const line of lines) {
    const t = line.trim();
    if (t === prev && t !== "" && !t.startsWith("#")) continue;
    prev = t;
    out.push(line);
  }
  return out.join("\n");
}

/** Dedupe repeated sentence units inside one long line (no newlines). */
function dedupeSentencesInChunk(text: string): string {
  const parts = text.split(/[。！？]/);
  const seen = new Set<string>();
  const out: string[] = [];
  for (let p of parts) {
    p = p.trim();
    if (p.length < 4) continue;
    if (seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out.join("。");
}

function sanitizeOneLine(line: string): string {
  let t = line.trim();
  if (!t) return "";
  if (isFluffLine(t)) return "";
  if (!t.startsWith("#") && t.length > 200) {
    t = dedupeSentencesInChunk(t);
    if (isFluffLine(t)) return "";
  }
  return stripBoldMarkers(t);
}

/**
 * Sanitize streamed slide outline before persisting / sending to PPTX API.
 */
export function sanitizeSlideOutlineMarkdown(raw: string): string {
  let s = raw.trim();
  if (!s) return s;

  s = dedupeConsecutiveLines(s);

  const lines = s.split(/\r?\n/);
  const kept: string[] = [];
  for (const line of lines) {
    const cleaned = sanitizeOneLine(line);
    if (cleaned === "") continue;
    kept.push(cleaned);
  }
  s = kept.join("\n").trim();

  s = s.replace(/\n{3,}/g, "\n\n");
  return s;
}
