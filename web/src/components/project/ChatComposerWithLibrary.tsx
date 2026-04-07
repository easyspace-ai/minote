import { ArrowUp, FileText, Paperclip, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export type LibraryDocRow = { id: number; original_name: string; file_size: number };

const MAX_ATTACH = 4;

function truncateLabel(name: string, max = 26): string {
  if (name.length <= max) return name;
  return `${name.slice(0, max - 1)}…`;
}

/** Map visible char offset (innerText, zwsp stripped) to offset in raw Text.data (may contain zwsp). */
function visibleOffsetToRaw(data: string, visibleOffset: number): number {
  let v = 0;
  for (let i = 0; i < data.length; i++) {
    if (data[i] === "\u200b") continue;
    if (v === visibleOffset) return i;
    v++;
  }
  return data.length;
}

/**
 * Build a DOM Range [start, end) over visible characters (aligned with innerText, zwsp stripped).
 */
function getRangeForVisibleIndices(el: HTMLElement, start: number, end: number): Range | null {
  const fullLen = el.innerText.replace(/\u200b/g, "").length;
  if (start < 0 || end > fullLen || start > end) return null;

  let acc = 0;
  const range = document.createRange();
  const w = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
  let n: Node | null;
  let started = false;

  while ((n = w.nextNode())) {
    const t = n as Text;
    const segLen = t.data.replace(/\u200b/g, "").length;
    const nodeStart = acc;
    const nodeEnd = acc + segLen;

    if (!started && start < nodeEnd) {
      const visInNode = start - nodeStart;
      range.setStart(t, visibleOffsetToRaw(t.data, visInNode));
      started = true;
    }
    if (started && end <= nodeEnd) {
      const visInNode = end - nodeStart;
      range.setEnd(t, visibleOffsetToRaw(t.data, visInNode));
      return range;
    }
    acc += segLen;
  }
  return null;
}

function createMentionElement(doc: LibraryDocRow): HTMLSpanElement {
  const label = truncateLabel(doc.original_name);
  const span = document.createElement("span");
  span.dataset.docId = String(doc.id);
  span.contentEditable = "false";
  span.className =
    "mention-pill align-middle inline-flex max-w-[min(100%,18rem)] cursor-default select-none items-center rounded-md border border-neutral-200 bg-neutral-100/95 px-1.5 py-0.5 text-[13px] font-medium text-neutral-800 shadow-sm dark:border-white/10 dark:bg-white/10 dark:text-neutral-100";
  span.textContent = `@${label}`;
  return span;
}

function insertAfterMention(span: HTMLSpanElement) {
  const zw = document.createElement("span");
  zw.className = "mention-zwsp";
  zw.appendChild(document.createTextNode("\u200b"));
  span.after(zw);
  const range = document.createRange();
  range.setStartAfter(zw);
  range.collapse(true);
  const sel = window.getSelection();
  if (sel) {
    sel.removeAllRanges();
    sel.addRange(range);
  }
}

export function ChatComposerWithLibrary(props: {
  docs: LibraryDocRow[];
  disabled?: boolean;
  busy?: boolean;
  placeholder?: string;
  onSubmit: (input: { text: string; documentIds: number[] }) => void | Promise<void>;
}) {
  const { docs, disabled, busy, placeholder, onSubmit } = props;
  const editorRef = useRef<HTMLDivElement>(null);
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionFilter, setMentionFilter] = useState("");
  const [hasContent, setHasContent] = useState(false);
  const [attachedIds, setAttachedIds] = useState<number[]>([]);
  const rootRef = useRef<HTMLDivElement>(null);

  const syncAttachedFromEditor = useCallback(() => {
    const el = editorRef.current;
    if (!el) return;
    const ids: number[] = [];
    el.querySelectorAll("[data-doc-id]").forEach((n) => {
      const id = Number((n as HTMLElement).dataset.docId);
      if (Number.isFinite(id)) ids.push(id);
    });
    setAttachedIds([...new Set(ids)]);
  }, []);

  const updateMentionFromPlainText = useCallback((value: string) => {
    const at = value.lastIndexOf("@");
    if (at < 0) {
      setMentionOpen(false);
      setMentionFilter("");
      return;
    }
    const after = value.slice(at + 1);
    if (after.includes(" ") || after.includes("\n")) {
      setMentionOpen(false);
      setMentionFilter("");
      return;
    }
    setMentionOpen(true);
    setMentionFilter(after);
  }, []);

  const handleEditorInput = useCallback(() => {
    const el = editorRef.current;
    if (!el) return;
    const value = el.innerText.replace(/\u200b/g, "");
    setHasContent(value.trim().length > 0);
    updateMentionFromPlainText(value);
    syncAttachedFromEditor();
  }, [syncAttachedFromEditor, updateMentionFromPlainText]);

  const docById = useMemo(() => {
    const m = new Map<number, LibraryDocRow>();
    for (const d of docs) m.set(d.id, d);
    return m;
  }, [docs]);

  const mentionCandidates = useMemo(() => {
    const q = mentionFilter.toLowerCase();
    return docs.filter((d) => {
      if (!q) return true;
      return d.original_name.toLowerCase().includes(q);
    });
  }, [docs, mentionFilter]);

  const pickDocument = useCallback(
    (doc: LibraryDocRow) => {
      const el = editorRef.current;
      if (!el) return;

      const existingSpans = el.querySelectorAll("[data-doc-id]");
      const already = el.querySelector(`[data-doc-id="${doc.id}"]`);
      if (existingSpans.length >= MAX_ATTACH && !already) return;

      const raw = el.innerText.replace(/\u200b/g, "");
      const at = raw.lastIndexOf("@");
      const inMention =
        at >= 0 &&
        !raw.slice(at + 1).includes(" ") &&
        !raw.slice(at + 1).includes("\n");

      const span = createMentionElement(doc);

      if (inMention) {
        const end = at + 1 + mentionFilter.length;
        const range = getRangeForVisibleIndices(el, at, end);
        if (range) {
          range.deleteContents();
          range.insertNode(span);
          insertAfterMention(span);
        } else {
          el.appendChild(span);
          insertAfterMention(span);
        }
      } else {
        el.appendChild(span);
        insertAfterMention(span);
      }

      setMentionOpen(false);
      setMentionFilter("");
      setHasContent(el.innerText.replace(/\u200b/g, "").trim().length > 0);
      syncAttachedFromEditor();

      requestAnimationFrame(() => {
        el.focus();
      });
    },
    [mentionFilter, syncAttachedFromEditor],
  );

  const removeAttachment = useCallback(
    (id: number) => {
      const el = editorRef.current;
      if (!el) return;
      el.querySelectorAll(`[data-doc-id="${id}"]`).forEach((n) => {
        const next = n.nextSibling;
        n.remove();
        if (next && next instanceof HTMLElement && next.classList?.contains("mention-zwsp")) {
          next.remove();
        }
      });
      handleEditorInput();
    },
    [handleEditorInput],
  );

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) {
        setMentionOpen(false);
      }
    }
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  const canSend =
    !disabled && !busy && (hasContent || attachedIds.length > 0);

  async function submit() {
    if (!canSend) return;
    const el = editorRef.current;
    if (!el) return;
    const docIds: number[] = [];
    el.querySelectorAll("[data-doc-id]").forEach((n) => {
      const id = Number((n as HTMLElement).dataset.docId);
      if (Number.isFinite(id)) docIds.push(id);
    });
    const unique = [...new Set(docIds)];
    const text = el.innerText.replace(/\u200b/g, "").trim();
    await onSubmit({ text, documentIds: unique });
    el.innerHTML = "";
    setHasContent(false);
    setAttachedIds([]);
    setMentionOpen(false);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void submit();
    }
    if (e.key === "Escape") setMentionOpen(false);
  }

  function onPaste(e: React.ClipboardEvent<HTMLDivElement>) {
    e.preventDefault();
    const t = e.clipboardData.getData("text/plain");
    document.execCommand("insertText", false, t);
  }

  return (
    <div ref={rootRef} className="p-6 pt-2 shrink-0">
      <div className="max-w-4xl mx-auto bg-white rounded-[2rem] shadow-xl border border-outline-variant/15 p-2 flex flex-col gap-2 group transition-all focus-within:border-outline-variant/40">
        {attachedIds.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 px-3 pt-2">
            {attachedIds.map((id) => {
              const d = docById.get(id);
              const label = d?.original_name ?? `#${id}`;
              return (
                <span
                  key={id}
                  className="inline-flex max-w-[min(100%,14rem)] items-center gap-1.5 rounded-lg border border-neutral-200/90 bg-neutral-50 px-2 py-1 text-xs font-medium text-neutral-800 shadow-sm dark:border-white/10 dark:bg-white/5 dark:text-neutral-100"
                >
                  <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <span className="truncate">{truncateLabel(label, 18)}</span>
                  <button
                    type="button"
                    className="shrink-0 rounded-full p-0.5 hover:bg-black/5 dark:hover:bg-white/10"
                    onClick={() => removeAttachment(id)}
                    aria-label="Remove attachment"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              );
            })}
          </div>
        )}

        <div className="relative flex items-end gap-1">
          <button
            type="button"
            className="mb-2 shrink-0 p-3 text-muted-foreground hover:text-black transition-colors"
            tabIndex={-1}
            aria-hidden
          >
            <Paperclip className="w-5 h-5" />
          </button>

          <div className="relative min-h-[52px] flex-1">
            {!hasContent && (
              <span className="pointer-events-none absolute left-2 top-2.5 text-base text-muted-foreground/50">
                {placeholder ?? "Ask anything…"}
              </span>
            )}
            <div
              ref={editorRef}
              role="textbox"
              aria-multiline="true"
              data-placeholder={placeholder}
              contentEditable={!(disabled || busy)}
              suppressContentEditableWarning
              onInput={handleEditorInput}
              onKeyDown={onKeyDown}
              onPaste={onPaste}
              className="min-h-[52px] w-full rounded-2xl border border-transparent bg-transparent px-2 py-2.5 text-base leading-relaxed text-black outline-none focus:border-outline-variant/25 focus:ring-0 empty:min-h-[52px] dark:text-neutral-100 [&_.mention-pill]:align-baseline"
              style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}
            />
          </div>

          <button
            type="button"
            disabled={!canSend}
            onClick={() => void submit()}
            className="mb-2 shrink-0 bg-black text-white w-10 h-10 rounded-full flex items-center justify-center hover:opacity-90 transition-all disabled:opacity-40 dark:bg-white dark:text-black"
          >
            <ArrowUp className="w-5 h-5" />
          </button>

          {mentionOpen && mentionCandidates.length > 0 && (
            <ul className="absolute bottom-full left-12 right-14 mb-2 max-h-48 overflow-y-auto rounded-xl border border-outline-variant/20 bg-popover py-1 text-sm shadow-sm dark:shadow-none dark:ring-1 dark:ring-white/10 z-50">
              {mentionCandidates.slice(0, 12).map((d) => (
                <li key={d.id}>
                  <button
                    type="button"
                    className="w-full px-3 py-2 text-left hover:bg-surface-container-high truncate"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => pickDocument(d)}
                  >
                    {d.original_name}
                    <span className="ml-2 text-muted-foreground text-xs">
                      {(d.file_size / 1024).toFixed(0)} KB
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <p className="px-4 pb-1 text-[10px] text-muted-foreground">
          输入 <kbd className="rounded border border-outline-variant/30 px-1">@</kbd> 选择资料：输入框内为灰色圆角标签（可多选）；上方为文件摘要。最多 {MAX_ATTACH} 个。
        </p>
      </div>
    </div>
  );
}
