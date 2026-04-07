import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Download, Loader2, Maximize2, Minimize2, X } from "lucide-react";
import { Streamdown } from "streamdown";
import type { StudioMaterial } from "@/api/chatclaw";
import { streamdownPlugins } from "@/core/streamdown";
import { downloadProjectPptx } from "@/lib/downloadProjectPptx";
import {
  downloadProjectStudioFile,
  fetchStudioFileBlob,
  fetchStudioFileText,
} from "@/lib/downloadProjectStudioFile";
import { cn } from "@/lib/utils";

function str(payload: Record<string, unknown>, key: string): string | undefined {
  const v = payload[key];
  return typeof v === "string" ? v : undefined;
}

function strList(payload: Record<string, unknown>, key: string): string[] {
  const v = payload[key];
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

function exportFileMeta(payload: Record<string, unknown>, key: string) {
  const raw = payload[key];
  if (!raw || typeof raw !== "object") return null;
  const o = raw as Record<string, unknown>;
  if (o.available !== true) return null;
  const fileName = typeof o.file_name === "string" ? o.file_name : "export.html";
  return { fileName };
}

/** sidebar：与 NotebookShell 右栏同宽内联；expanded：居中放大层 */
export type StudioMaterialContentLayout = "default" | "sidebar" | "expanded" | "embedded";

/** Payload text that is clearly HTML → iframe preview; otherwise treat as Markdown. */
function studioPayloadLooksLikeHtml(s: string): boolean {
  const t = s.trim();
  if (!t) return false;
  if (/^<!DOCTYPE\s+/i.test(t)) return true;
  if (/^<html[\s>]/i.test(t)) return true;
  const firstLine = (t.split("\n")[0] ?? "").trim();
  if (firstLine.length > 120) return false;
  if (/^<\/?[a-z][\w-]*(\s[\s\S]*?)?(\/>|>)$/i.test(firstLine)) {
    return /<\/[a-z][\w-]*>/i.test(t) || /\/>/.test(t);
  }
  return false;
}

function studioMarkdownScrollClass(contentLayout: StudioMaterialContentLayout): string {
  switch (contentLayout) {
    case "sidebar":
      return "max-h-[min(52vh,560px)]";
    case "expanded":
      return "max-h-[min(70vh,720px)]";
    case "embedded":
      return "max-h-[calc(100dvh-15rem)] min-h-[120px]";
    default:
      return "max-h-[min(60vh,520px)]";
  }
}

/** Rendered Markdown (GFM + math + raw HTML snippets via rehype-raw). */
function StudioMarkdownPreview(props: {
  markdown: string;
  contentLayout: StudioMaterialContentLayout;
}) {
  const { markdown, contentLayout } = props;
  return (
    <div
      className={cn(
        "overflow-y-auto overflow-x-hidden pr-1",
        studioMarkdownScrollClass(contentLayout),
        "[&_a]:text-primary [&_a]:underline [&_a]:underline-offset-2",
        "[&_pre]:overflow-x-auto [&_pre]:rounded-lg [&_pre]:bg-black/[0.04] [&_pre]:p-3 [&_pre]:text-xs",
        "[&_table]:block [&_table]:max-w-full [&_table]:overflow-x-auto [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1",
        "[&_img]:max-w-full [&_img]:rounded-lg",
      )}
    >
      <Streamdown className="text-sm leading-relaxed text-on-surface" {...streamdownPlugins}>
        {markdown}
      </Streamdown>
    </div>
  );
}

function StudioMarkdownWithSourceToggle(props: {
  markdown: string;
  contentLayout: StudioMaterialContentLayout;
}) {
  const { markdown, contentLayout } = props;
  return (
    <div className="space-y-3">
      <StudioMarkdownPreview markdown={markdown} contentLayout={contentLayout} />
      <details className="rounded-xl border border-outline-variant/10 bg-surface-container-low/50 p-3">
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground">
          原始 Markdown
        </summary>
        <pre className="mt-2 max-h-[min(36vh,320px)] overflow-auto whitespace-pre-wrap text-xs leading-relaxed text-on-surface">
          {markdown}
        </pre>
      </details>
    </div>
  );
}

function StudioAudioMaterialBody(props: {
  projectId?: number | null;
  materialId: number;
  payload: Record<string, unknown>;
  contentLayout?: StudioMaterialContentLayout;
}) {
  const { projectId, materialId, payload, contentLayout = "default" } = props;
  const extUrl = str(payload, "audioUrl") ?? str(payload, "url");
  const serverFile = str(payload, "file_name");
  const transcript = str(payload, "markdown");
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [dlBusy, setDlBusy] = useState(false);

  useEffect(() => {
    if (extUrl || !serverFile || projectId == null || projectId <= 0) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    setLoadErr(null);
    void fetchStudioFileBlob(projectId, materialId)
      .then((blob) => {
        if (!cancelled) setBlobUrl(URL.createObjectURL(blob));
      })
      .catch((e: unknown) => {
        if (!cancelled) setLoadErr(e instanceof Error ? e.message : "加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
      setBlobUrl((u) => {
        if (u) URL.revokeObjectURL(u);
        return null;
      });
    };
  }, [projectId, materialId, serverFile, extUrl]);

  if (extUrl) {
    return (
      <div className="space-y-3">
        <audio className="w-full" controls src={extUrl} />
        {str(payload, "note") ? <p className="text-sm text-muted-foreground">{str(payload, "note")}</p> : null}
      </div>
    );
  }

  if (serverFile && projectId != null && projectId > 0) {
    const dlName = serverFile.endsWith(".mp3") ? serverFile : `${serverFile}.mp3`;
    return (
      <div className="space-y-4">
        <div className="space-y-2">
          <button
            type="button"
            disabled={dlBusy}
            onClick={() => {
              setDlBusy(true);
              void downloadProjectStudioFile(projectId, materialId, dlName)
                .catch((e: unknown) => console.error(e))
                .finally(() => setDlBusy(false));
            }}
            className="inline-flex items-center gap-2 rounded-xl border border-outline-variant/20 bg-surface-container-low px-4 py-2 text-sm font-semibold text-on-surface hover:bg-surface-container-high disabled:opacity-50"
          >
            {dlBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
            下载音频
          </button>
        </div>
        {loading ? (
          <div className="text-muted-foreground flex items-center gap-2 text-sm">
            <Loader2 className="h-4 w-4 animate-spin" /> 加载播放器…
          </div>
        ) : null}
        {loadErr ? <p className="text-sm text-destructive">{loadErr}</p> : null}
        {blobUrl ? (
          <audio
            className={cn(
              "w-full",
              contentLayout === "expanded" && "max-w-full",
            )}
            controls
            src={blobUrl}
          />
        ) : null}
        {transcript ? (
          <details
            className={cn(
              "rounded-xl border border-outline-variant/10 bg-surface-container-low/50 p-3",
              contentLayout === "sidebar" && "max-h-[min(40vh,400px)] overflow-auto",
              contentLayout === "expanded" && "max-h-[min(50vh,520px)] overflow-auto",
            )}
          >
            <summary className="cursor-pointer text-xs font-semibold text-muted-foreground">口播稿</summary>
            <div className="mt-2 max-h-[min(32vh,320px)] overflow-y-auto">
              <Streamdown
                className="text-xs leading-relaxed text-on-surface"
                {...streamdownPlugins}
              >
                {transcript}
              </Streamdown>
            </div>
          </details>
        ) : null}
      </div>
    );
  }

  return <p className="text-sm text-muted-foreground">暂无音频：需要 audioUrl/url，或服务端保存的 studio 音频文件。</p>;
}

function StudioExportPreview(props: {
  projectId: number;
  materialId: number;
  downloadLabel: string;
  downloadName: string;
  markdownFallback?: string;
  contentLayout?: StudioMaterialContentLayout;
}) {
  const {
    projectId,
    materialId,
    downloadLabel,
    downloadName,
    markdownFallback,
    contentLayout = "default",
  } = props;
  const iframeClass =
    contentLayout === "expanded"
      ? "h-[min(78vh,820px)] w-full shrink-0 rounded-xl border border-black/[0.08]"
      : contentLayout === "sidebar"
        ? "h-[min(40vh,360px)] w-full shrink-0 rounded-xl border border-black/[0.08]"
        : contentLayout === "embedded"
          ? "h-[calc(100dvh-15rem)] min-h-[280px] w-full shrink-0 rounded-xl border border-black/[0.08]"
          : "h-[min(60vh,520px)] w-full rounded-xl border border-outline-variant/15";
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [srcDoc, setSrcDoc] = useState<string | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setLoadErr(null);
    void fetchStudioFileText(projectId, materialId)
      .then((t) => {
        if (!cancelled) setSrcDoc(t);
      })
      .catch((e: unknown) => {
        if (!cancelled) setLoadErr(e instanceof Error ? e.message : "预览加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, materialId]);

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            setErr(null);
            setBusy(true);
            void downloadProjectStudioFile(projectId, materialId, downloadName)
              .catch((e: unknown) => {
                console.error(e);
                setErr(e instanceof Error ? e.message : "下载失败");
              })
              .finally(() => setBusy(false));
          }}
          className="inline-flex items-center gap-2 rounded-xl border border-outline-variant/20 bg-surface-container-low px-4 py-2 text-sm font-semibold text-on-surface hover:bg-surface-container-high disabled:opacity-50"
        >
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
          {downloadLabel}
        </button>
        {err ? <p className="text-sm text-destructive">{err}</p> : null}
      </div>
      {loading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" /> 加载预览…
        </div>
      ) : loadErr ? (
        <p className="text-sm text-destructive">{loadErr}</p>
      ) : srcDoc ? (
        <iframe
          title="Studio export preview"
          className={iframeClass}
          srcDoc={srcDoc}
          sandbox="allow-scripts allow-same-origin"
        />
      ) : null}
      {markdownFallback ? (
        <details className="rounded-xl border border-outline-variant/10 bg-surface-container-low/50 p-3">
          <summary className="cursor-pointer text-xs font-semibold text-muted-foreground">
            生成用 Markdown / 源码
          </summary>
          <div className="mt-2 max-h-[min(40vh,400px)] overflow-y-auto">
            {studioPayloadLooksLikeHtml(markdownFallback) ? (
              <iframe
                title="Source HTML preview"
                className="h-[min(36vh,360px)] w-full rounded-lg border border-black/[0.08]"
                srcDoc={markdownFallback}
                sandbox="allow-scripts allow-same-origin"
              />
            ) : (
              <Streamdown
                className="text-sm leading-relaxed text-on-surface"
                {...streamdownPlugins}
              >
                {markdownFallback}
              </Streamdown>
            )}
          </div>
        </details>
      ) : null}
    </div>
  );
}

type PreviewPaneProps = {
  material: StudioMaterial;
  projectId?: number | null;
  /** sidebar：NotebookShell 右栏内替换 Studio（与侧栏同宽、无全屏遮罩）；expanded：居中放大层；embedded：中间栏全高 */
  variant: "sidebar" | "expanded" | "embedded";
  onClose: () => void;
  /** 侧栏：内容区放大 */
  onExpand?: () => void;
  /** 放大层：缩回侧栏 */
  onMinimize?: () => void;
};

/**
 * 正文 + 标题栏（侧栏内联：标题栏「缩起」= 返回 Studio 列表；「放大」= 居中浮层）。
 */
export function StudioMaterialPreviewPane({
  material,
  projectId,
  variant,
  onClose,
  onExpand,
  onMinimize,
}: PreviewPaneProps) {
  const p = material.payload ?? {};
  const contentLayout: StudioMaterialContentLayout =
    variant === "expanded" ? "expanded" : variant === "sidebar" ? "sidebar" : "embedded";

  const scrollBox =
    variant === "expanded"
      ? "min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-5 py-4"
      : variant === "sidebar"
        ? "relative min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-3 py-2 sm:px-4 sm:py-3"
        : "flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden px-6 py-4";

  const isSidebar = variant === "sidebar";
  const isExpanded = variant === "expanded";

  return (
    <div
      className={cn(
        "flex min-h-0 flex-col overflow-hidden text-on-surface",
        isSidebar && "h-full min-h-0 flex-1",
        isExpanded && "h-full min-h-0 bg-white",
        variant === "embedded" && "h-full min-h-0 flex-1 bg-white",
      )}
    >
      <div
        className={cn(
          "flex shrink-0 items-start justify-between gap-2 border-b border-black/[0.08] px-3 py-2.5 sm:px-4 sm:py-3",
          isExpanded && "px-5 py-4",
        )}
      >
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
            Studio
            {material.kind ? (
              <span className="text-muted-foreground/80 font-normal"> · {material.kind}</span>
            ) : null}
          </p>
          <h2
            className={cn(
              "mt-0.5 font-bold tracking-tight text-foreground",
              isExpanded ? "text-lg" : "truncate text-sm sm:text-base",
            )}
          >
            {material.title}
          </h2>
          {material.subtitle ? (
            <p className="mt-1 text-xs text-muted-foreground">{material.subtitle}</p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          {isExpanded ? (
            <>
              <button
                type="button"
                onClick={onMinimize}
                className="rounded-lg p-2 text-muted-foreground hover:bg-black/[0.06] hover:text-foreground"
                aria-label="缩回侧栏预览"
                title="缩回侧栏"
              >
                <Minimize2 className="h-5 w-5" strokeWidth={2} />
              </button>
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg p-2 text-muted-foreground hover:bg-black/[0.06] hover:text-foreground"
                aria-label="关闭"
                title="关闭"
              >
                <X className="h-5 w-5" />
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-2 text-muted-foreground hover:bg-black/[0.06] hover:text-foreground"
              aria-label="返回 Studio"
              title="返回 Studio"
            >
              <Minimize2 className="h-5 w-5" strokeWidth={2} />
            </button>
          )}
        </div>
      </div>
      <div className={cn(scrollBox, "flex min-h-0 flex-col")}>
        {isSidebar && onExpand ? (
          <div className="mb-2 flex justify-end">
            <button
              type="button"
              onClick={onExpand}
              className="inline-flex items-center gap-1.5 rounded-lg border border-black/[0.08] bg-white px-2.5 py-1.5 text-xs font-semibold text-foreground shadow-sm hover:bg-black/[0.03]"
              title="放大预览"
              aria-label="放大预览"
            >
              <Maximize2 className="h-3.5 w-3.5" strokeWidth={2} />
              放大
            </button>
          </div>
        ) : null}
        <div className="min-h-0 flex-1">
          <MaterialBody
            projectId={projectId}
            materialId={material.id}
            kind={material.kind}
            payload={p}
            contentLayout={contentLayout}
          />
        </div>
      </div>
    </div>
  );
}

type ExpandedOverlayProps = {
  projectId?: number | null;
  material: StudioMaterial | null;
  open: boolean;
  /** 缩回右栏内联预览 */
  onMinimize: () => void;
  /** 完全关闭（侧栏详情一并关掉） */
  onCloseAll: () => void;
};

/** 仅居中放大层（侧栏内点击「放大」后出现）；侧栏本身仍在 NotebookShell 内。 */
export function StudioMaterialExpandedOverlay({
  projectId,
  material,
  open,
  onMinimize,
  onCloseAll,
}: ExpandedOverlayProps) {
  useEffect(() => {
    if (!open || !material) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.preventDefault();
      onMinimize();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, material, onMinimize]);

  if (!material || !open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[210] flex items-center justify-center bg-black/45 p-3 sm:p-6"
      role="presentation"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        onClick={onMinimize}
        aria-label="返回侧栏"
      />
      <div
        className="relative z-[1] flex max-h-[92vh] w-full max-w-[920px] flex-col overflow-hidden rounded-2xl border border-black/10 bg-white shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <StudioMaterialPreviewPane
          variant="expanded"
          material={material}
          projectId={projectId}
          onClose={onCloseAll}
          onMinimize={onMinimize}
        />
      </div>
    </div>,
    document.body,
  );
}

function MaterialBody(props: {
  projectId?: number | null;
  materialId: number;
  kind: string;
  payload: Record<string, unknown>;
  contentLayout?: StudioMaterialContentLayout;
}) {
  const { projectId, materialId, kind, payload, contentLayout = "default" } = props;
  const iframeHtml =
    contentLayout === "expanded"
      ? "h-[min(72vh,720px)] w-full rounded-xl border border-black/[0.08]"
      : contentLayout === "sidebar"
        ? "h-[min(38vh,360px)] w-full rounded-xl border border-black/[0.08]"
        : contentLayout === "embedded"
          ? "h-[calc(100dvh-15rem)] min-h-[280px] w-full shrink-0 rounded-xl border border-black/[0.08]"
          : "h-[min(60vh,480px)] w-full rounded-xl border border-outline-variant/15";
  const [pptxBusy, setPptxBusy] = useState(false);
  const [pptxErr, setPptxErr] = useState<string | null>(null);

  switch (kind) {
    case "audio": {
      return (
        <StudioAudioMaterialBody
          projectId={projectId}
          materialId={materialId}
          payload={payload}
          contentLayout={contentLayout}
        />
      );
    }
    case "slides": {
      const urls = strList(payload, "slideUrls");
      const slideMd = str(payload, "markdown") ?? str(payload, "text");
      const pptx = payload["pptx"];
      const pptxObj = pptx && typeof pptx === "object" ? (pptx as Record<string, unknown>) : null;
      const serverPptxName = str(payload, "file_name");
      const pptxOk =
        pptxObj?.available === true ||
        (typeof serverPptxName === "string" && /\.pptx$/i.test(serverPptxName));
      const pptxName =
        (typeof pptxObj?.file_name === "string" ? pptxObj.file_name : null) ??
        serverPptxName ??
        "presentation.pptx";

      const pptxDownload =
        projectId != null && projectId > 0 && pptxOk ? (
          <div className="space-y-2">
            <button
              type="button"
              disabled={pptxBusy}
              onClick={() => {
                setPptxErr(null);
                setPptxBusy(true);
                void downloadProjectPptx(projectId, materialId, pptxName)
                  .catch((e: unknown) => {
                    console.error(e);
                    setPptxErr(e instanceof Error ? e.message : "下载失败");
                  })
                  .finally(() => setPptxBusy(false));
              }}
              className="inline-flex items-center gap-2 rounded-xl border border-outline-variant/20 bg-surface-container-low px-4 py-2 text-sm font-semibold text-on-surface hover:bg-surface-container-high disabled:opacity-50"
            >
              {pptxBusy ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              下载 PPTX
            </button>
            {pptxErr ? <p className="text-sm text-destructive">{pptxErr}</p> : null}
          </div>
        ) : null;

      if (urls.length === 0 && slideMd) {
        return (
          <div className="space-y-4">
            {pptxDownload}
            <StudioMarkdownWithSourceToggle markdown={slideMd} contentLayout={contentLayout} />
          </div>
        );
      }
      if (urls.length === 0) {
        return (
          <div className="space-y-4">
            {pptxDownload}
            <p className="text-sm leading-relaxed text-muted-foreground">
              当前条目没有保存幻灯片大纲（Markdown）。若已生成过文件，可先下载 PPTX；新创建的幻灯片在服务端会同时保存大纲供此处预览。
            </p>
          </div>
        );
      }
      return (
        <div className="space-y-4">
          {pptxDownload}
          {slideMd ? (
            <StudioMarkdownWithSourceToggle markdown={slideMd} contentLayout={contentLayout} />
          ) : null}
          {urls.map((u, i) => (
            <img
              key={i}
              src={u}
              alt={`Slide ${i + 1}`}
              className={cn(
                "w-full rounded-xl border",
                contentLayout === "sidebar"
                  ? "max-h-[min(44vh,480px)] border-black/[0.08] object-contain"
                  : contentLayout === "expanded"
                    ? "max-h-[min(70vh,800px)] border-black/[0.08] object-contain"
                    : "border-outline-variant/15",
              )}
            />
          ))}
        </div>
      );
    }
    case "html": {
      const ex = exportFileMeta(payload, "html");
      const md = str(payload, "markdown");
      const serverFile = str(payload, "file_name");
      if (projectId != null && projectId > 0 && (ex || serverFile)) {
        return (
          <StudioExportPreview
            projectId={projectId}
            materialId={materialId}
            downloadLabel="下载 HTML"
            downloadName={ex?.fileName ?? serverFile ?? "page.html"}
            markdownFallback={md}
            contentLayout={contentLayout}
          />
        );
      }
      const src = str(payload, "iframeUrl") ?? str(payload, "url");
      const srcDocInline = str(payload, "srcDoc");
      if (src) {
        return <iframe title="HTML preview" className={iframeHtml} src={src} />;
      }
      if (srcDocInline) {
        return <iframe title="HTML preview" className={iframeHtml} srcDoc={srcDocInline} />;
      }
      if (md) {
        if (studioPayloadLooksLikeHtml(md)) {
          return (
            <iframe
              title="HTML preview"
              className={iframeHtml}
              srcDoc={md}
              sandbox="allow-scripts allow-same-origin"
            />
          );
        }
        return <StudioMarkdownWithSourceToggle markdown={md} contentLayout={contentLayout} />;
      }
      return <p className="text-sm text-muted-foreground">Add iframeUrl, srcDoc, or markdown in payload.</p>;
    }
    case "mindmap": {
      const ex = exportFileMeta(payload, "mindmap");
      const md = str(payload, "markdown");
      const serverFile = str(payload, "file_name");
      if (projectId != null && projectId > 0 && (ex || serverFile)) {
        return (
          <StudioExportPreview
            projectId={projectId}
            materialId={materialId}
            downloadLabel="下载思维导图 HTML"
            downloadName={ex?.fileName ?? serverFile ?? "mindmap.html"}
            markdownFallback={md}
            contentLayout={contentLayout}
          />
        );
      }
      const raw = payload["nodes"];
      if (raw != null && typeof raw === "object") {
        return (
          <pre className="overflow-x-auto rounded-xl bg-surface-container-low p-3 text-xs leading-relaxed">
            {JSON.stringify(raw, null, 2)}
          </pre>
        );
      }
      if (md) {
        if (studioPayloadLooksLikeHtml(md)) {
          return (
            <iframe
              title="Mind map HTML"
              className={iframeHtml}
              srcDoc={md}
              sandbox="allow-scripts allow-same-origin"
            />
          );
        }
        return <StudioMarkdownWithSourceToggle markdown={md} contentLayout={contentLayout} />;
      }
      return <p className="text-sm text-muted-foreground">Mind map needs markdown text or server export.</p>;
    }
    case "report":
    case "infographic":
    case "quiz":
    case "data_table":
    default: {
      const md = str(payload, "markdown") ?? str(payload, "text");
      if (md) {
        if (studioPayloadLooksLikeHtml(md)) {
          return (
            <iframe
              title="Studio material HTML"
              className={iframeHtml}
              srcDoc={md}
              sandbox="allow-scripts allow-same-origin"
            />
          );
        }
        return <StudioMarkdownWithSourceToggle markdown={md} contentLayout={contentLayout} />;
      }
      return (
        <pre className="overflow-x-auto text-xs leading-relaxed text-muted-foreground">
          {JSON.stringify(payload, null, 2)}
        </pre>
      );
    }
  }
}
