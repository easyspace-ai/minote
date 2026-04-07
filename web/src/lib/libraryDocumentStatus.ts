/** Matches backend document.Status* constants. */
export const DOC_STATUS = {
  pending: 0,
  processing: 1,
  completed: 2,
  failed: 3,
} as const;

export type LibraryDocumentRow = {
  id: number;
  original_name: string;
  file_size: number;
  /** From API `starred`; default false if omitted. */
  starred?: boolean;
  parsing_status: number;
  parsing_progress: number;
  parsing_error?: string;
  embedding_status: number;
  embedding_progress: number;
  embedding_error?: string;
  split_total: number;
  word_total?: number;
};

export function isLibraryDocumentProcessing(d: LibraryDocumentRow): boolean {
  if (d.parsing_status === DOC_STATUS.failed || d.embedding_status === DOC_STATUS.failed) {
    return false;
  }
  return !(d.parsing_status === DOC_STATUS.completed && d.embedding_status === DOC_STATUS.completed);
}

/** Single 0–100 bar: parse ~ first half, embedding ~ second half (matches backend phases). */
export function libraryDocumentCombinedProgress(d: LibraryDocumentRow): number {
  const pp = Math.min(100, Math.max(0, d.parsing_progress ?? 0));
  const ep = Math.min(100, Math.max(0, d.embedding_progress ?? 0));
  if (d.parsing_status === DOC_STATUS.failed || d.embedding_status === DOC_STATUS.failed) {
    return 0;
  }
  if (d.embedding_status === DOC_STATUS.completed) {
    return 100;
  }
  if (d.parsing_status === DOC_STATUS.completed) {
    if (d.embedding_status === DOC_STATUS.pending) {
      return 52;
    }
    return 50 + (ep / 100) * 50;
  }
  if (d.parsing_status === DOC_STATUS.processing) {
    return (pp / 100) * 50;
  }
  return 5;
}

export function libraryDocumentStatusParts(d: LibraryDocumentRow): {
  line: string;
  detail?: string;
} {
  if (d.parsing_status === DOC_STATUS.failed) {
    return { line: "解析失败", detail: d.parsing_error || undefined };
  }
  if (d.embedding_status === DOC_STATUS.failed) {
    return { line: "嵌入失败", detail: d.embedding_error || undefined };
  }
  if (d.parsing_status === DOC_STATUS.completed && d.embedding_status === DOC_STATUS.completed) {
    return { line: "可检索" };
  }
  if (d.parsing_status === DOC_STATUS.processing) {
    return { line: "解析中…" };
  }
  if (d.parsing_status === DOC_STATUS.completed) {
    return { line: "嵌入与分块…" };
  }
  return { line: "排队处理…" };
}
