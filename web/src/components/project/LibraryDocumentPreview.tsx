import { useEffect, useRef, useState, type ReactNode } from "react";
import { Download, Loader2, X } from "lucide-react";
import { Streamdown } from "streamdown";
import { chatclawApi, type DocumentChatAttachment } from "@/api/chatclaw";
import { streamdownPlugins } from "@/core/streamdown";
import type { LibraryDocumentRow } from "@/lib/libraryDocumentStatus";
import { cn } from "@/lib/utils";

/** Avoid decoding huge payloads in the browser (chat-attachment returns full file). */
export const LIBRARY_PREVIEW_MAX_BYTES = 18 * 1024 * 1024;

function base64ToBlob(base64: string, mime: string): Blob {
  const binary = atob(base64);
  const len = binary.length;
  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i++) bytes[i] = binary.charCodeAt(i);
  return new Blob([bytes], { type: mime || "application/octet-stream" });
}

function effectiveMime(originalName: string, mime: string): string {
  const m = (mime || "").split(";")[0]!.trim().toLowerCase();
  if (m && m !== "application/octet-stream") return mime.split(";")[0]!.trim();
  const lower = originalName.toLowerCase();
  if (lower.endsWith(".pdf")) return "application/pdf";
  if (lower.endsWith(".md") || lower.endsWith(".markdown")) return "text/markdown";
  if (lower.endsWith(".html") || lower.endsWith(".htm")) return "text/html";
  if (lower.endsWith(".txt")) return "text/plain";
  if (lower.endsWith(".json")) return "application/json";
  if (lower.endsWith(".png")) return "image/png";
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  if (lower.endsWith(".gif")) return "image/gif";
  if (lower.endsWith(".webp")) return "image/webp";
  if (lower.endsWith(".svg")) return "image/svg+xml";
  if (lower.endsWith(".mp3")) return "audio/mpeg";
  if (lower.endsWith(".wav")) return "audio/wav";
  if (lower.endsWith(".mp4")) return "video/mp4";
  return "application/octet-stream";
}

function isTextualMime(mime: string): boolean {
  const m = mime.toLowerCase();
  return (
    m.startsWith("text/") ||
    m === "application/json" ||
    m === "application/xml" ||
    m.endsWith("+json") ||
    m.endsWith("+xml")
  );
}

type PaneProps = {
  doc: LibraryDocumentRow;
  onClose: () => void;
  /** 嵌入中间栏：占满高度；false 用于弹窗内固定视口高度 */
  embedded?: boolean;
  className?: string;
};

/**
 * 资料正文预览（中间栏或弹窗内复用）。
 */
export function LibraryDocumentPreviewPane({
  doc,
  onClose,
  embedded = false,
  className,
}: PaneProps) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [attachment, setAttachment] = useState<DocumentChatAttachment | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    setErr(null);
    setAttachment(null);
    void chatclawApi.documents
      .chatAttachment(doc.id)
      .then((data) => {
        if (!cancelled) setAttachment(data);
      })
      .catch((e: unknown) => {
        if (!cancelled) setErr(e instanceof Error ? e.message : "加载失败");
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [doc.id]);

  useEffect(() => {
    if (!attachment?.base64_data || attachment.file_size > LIBRARY_PREVIEW_MAX_BYTES) {
      setPreviewUrl(null);
      return;
    }
    const mime = effectiveMime(attachment.original_name, attachment.mime_type);
    const needsObjectUrl =
      mime.startsWith("image/") ||
      mime === "application/pdf" ||
      mime.startsWith("audio/") ||
      mime.startsWith("video/") ||
      mime === "text/html";
    if (!needsObjectUrl) {
      setPreviewUrl(null);
      return;
    }
    let url: string | null = null;
    try {
      const blob = base64ToBlob(attachment.base64_data, mime);
      url = URL.createObjectURL(blob);
      setPreviewUrl(url);
    } catch {
      setErr("无法解析文件数据");
      setPreviewUrl(null);
    }
    return () => {
      if (url) URL.revokeObjectURL(url);
    };
  }, [attachment]);

  const downloadBlob = () => {
    if (!attachment?.base64_data) return;
    const mime = effectiveMime(attachment.original_name, attachment.mime_type);
    const blob = base64ToBlob(attachment.base64_data, mime);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = attachment.original_name || "document";
    a.click();
    URL.revokeObjectURL(url);
  };

  const mime = attachment
    ? effectiveMime(attachment.original_name, attachment.mime_type)
    : "";
  const tooLarge = attachment != null && attachment.file_size > LIBRARY_PREVIEW_MAX_BYTES;

  const scrollBox = embedded
    ? "flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden px-6 py-4"
    : "min-h-0 flex-1 overflow-y-auto px-5 py-4";
  const imgShell = embedded
    ? "flex min-h-0 flex-1 justify-center overflow-auto"
    : "flex max-h-[min(70vh,640px)] justify-center overflow-auto";
  const imgClass = embedded
    ? "max-h-full max-w-full object-contain"
    : "max-h-[min(70vh,640px)] max-w-full object-contain";
  const iframeFrame = embedded
    ? "h-[calc(100dvh-14rem)] min-h-[360px] w-full shrink-0 rounded-xl border border-black/[0.08]"
    : "h-[min(72vh,680px)] w-full rounded-xl border border-outline-variant/15";
  const iframeHtml = embedded
    ? "h-[calc(100dvh-15rem)] min-h-[280px] w-full shrink-0 rounded-xl border border-black/[0.08]"
    : "h-[min(72vh,680px)] w-full rounded-xl border border-outline-variant/15";
  const mdBox = embedded
    ? "min-h-0 flex-1 overflow-y-auto pr-1"
    : "max-h-[min(70vh,640px)] overflow-y-auto pr-1";
  const preBox = embedded
    ? "min-h-0 flex-1 overflow-auto whitespace-pre-wrap wrap-break-word rounded-xl border border-black/[0.08] bg-muted/30 p-4 text-xs leading-relaxed"
    : "max-h-[min(70vh,640px)] overflow-auto whitespace-pre-wrap wrap-break-word rounded-xl border border-outline-variant/10 bg-muted/30 p-4 text-xs leading-relaxed";
  const videoClass = embedded ? "max-h-full w-full rounded-xl" : "max-h-[min(60vh,520px)] w-full rounded-xl";

  let body: ReactNode = null;
  if (busy) {
    body = (
      <div className="text-muted-foreground flex items-center gap-2 py-12 text-sm">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载预览…
      </div>
    );
  } else if (err) {
    body = <p className="text-destructive py-6 text-sm">{err}</p>;
  } else if (!attachment?.base64_data) {
    body = <p className="text-muted-foreground py-6 text-sm">没有可预览的内容。</p>;
  } else if (tooLarge) {
    body = (
      <div className="space-y-3 py-4 text-sm">
        <p className="text-muted-foreground">
          文件较大（约 {(attachment.file_size / (1024 * 1024)).toFixed(1)} MB），为避免浏览器卡顿，不在此内联预览。
        </p>
        <button
          type="button"
          onClick={downloadBlob}
          className="inline-flex items-center gap-2 rounded-xl border border-outline-variant/20 bg-surface-container-low px-4 py-2 font-semibold text-on-surface hover:bg-surface-container-high"
        >
          <Download className="h-4 w-4" />
          下载到本地查看
        </button>
      </div>
    );
  } else if (mime.startsWith("image/")) {
    const src = previewUrl;
    body = src ? (
      <div className={embedded ? cn(imgShell, "flex-1") : imgShell}>
        <img src={src} alt={attachment.original_name} className={imgClass} />
      </div>
    ) : null;
  } else if (mime === "application/pdf") {
    const src = previewUrl;
    body = src ? <iframe title={attachment.original_name} src={src} className={iframeFrame} /> : null;
  } else if (mime.startsWith("audio/")) {
    const src = previewUrl;
    body = src ? <audio className="w-full" controls src={src} /> : null;
  } else if (mime.startsWith("video/")) {
    const src = previewUrl;
    body = src ? <video className={videoClass} controls src={src} /> : null;
  } else if (mime === "text/html") {
    const src = previewUrl;
    body = src ? (
      <iframe title={attachment.original_name} src={src} sandbox="" className={iframeHtml} />
    ) : null;
  } else if (isTextualMime(mime) || mime === "application/octet-stream") {
    try {
      const bytes = Uint8Array.from(atob(attachment.base64_data), (c) => c.charCodeAt(0));
      const text = new TextDecoder("utf-8", { fatal: false }).decode(bytes);
      const looksBinary = text.includes("\0") && mime === "application/octet-stream";
      if (looksBinary) {
        body = (
          <p className="text-muted-foreground text-sm">
            二进制文件无法在浏览器中预览，请使用上方下载。
          </p>
        );
      } else if (mime === "text/markdown" || /\.(md|markdown)$/i.test(attachment.original_name)) {
        body = (
          <div className={mdBox}>
            <Streamdown className="text-sm leading-relaxed" {...streamdownPlugins}>
              {text}
            </Streamdown>
          </div>
        );
      } else {
        body = <pre className={preBox}>{text}</pre>;
      }
    } catch {
      body = <p className="text-destructive text-sm">无法解码文本内容。</p>;
    }
  } else {
    body = (
      <p className="text-muted-foreground text-sm">
        不支持在线预览此类型（{mime || "未知"}），请下载查看。
      </p>
    );
  }

  const bodyWrap = embedded ? (
    <div className="flex min-h-0 flex-1 flex-col">{body}</div>
  ) : (
    body
  );

  return (
    <div
      className={cn(
        "flex min-h-0 flex-col overflow-hidden bg-white text-on-surface",
        embedded && "h-full min-h-0 flex-1",
        className,
      )}
    >
      <div
        className={cn(
          "flex shrink-0 items-start justify-between gap-4 border-b border-black/[0.06] px-6 py-3",
          !embedded && "border-outline-variant/10 px-5 py-4",
        )}
      >
        <div className="min-w-0">
          <p className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">资料预览</p>
          <h2 className="mt-1 truncate text-base font-bold tracking-tight text-foreground md:text-lg">
            {doc.original_name}
          </h2>
          {attachment && !tooLarge ? (
            <p className="mt-1 text-xs text-muted-foreground">
              {(attachment.file_size / 1024).toFixed(1)} KB · {mime}
            </p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {attachment?.base64_data ? (
            <button
              type="button"
              onClick={downloadBlob}
              className="rounded-lg p-2 text-muted-foreground hover:bg-black/[0.06] hover:text-foreground"
              title="下载"
              aria-label="下载"
            >
              <Download className="h-5 w-5" />
            </button>
          ) : null}
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-muted-foreground hover:bg-black/[0.06] hover:text-foreground"
            aria-label="关闭预览"
            title="关闭预览"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>
      <div className={scrollBox}>{bodyWrap}</div>
    </div>
  );
}

type DialogProps = {
  doc: LibraryDocumentRow | null;
  open: boolean;
  onClose: () => void;
};

/** 可选：独立弹窗预览（与中间栏预览二选一或并存） */
export function LibraryDocumentPreviewDialog({ doc, open, onClose }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open) {
      if (!d.open) d.showModal();
    } else if (d.open) {
      d.close();
    }
  }, [open]);

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    const onCancel = (e: Event) => {
      e.preventDefault();
      onClose();
    };
    d.addEventListener("cancel", onCancel);
    return () => d.removeEventListener("cancel", onCancel);
  }, [onClose]);

  if (!doc) return null;

  return (
    <dialog
      ref={ref}
      className="max-h-[92vh] w-[min(100vw-2rem,900px)] rounded-2xl border border-outline-variant/15 bg-white p-0 text-on-surface shadow-xl backdrop:bg-black/40"
      onClick={(e) => {
        if (e.target === ref.current) onClose();
      }}
    >
      <div className="flex max-h-[92vh] flex-col">
        {open ? <LibraryDocumentPreviewPane doc={doc} onClose={onClose} embedded={false} /> : null}
      </div>
    </dialog>
  );
}
