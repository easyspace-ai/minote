"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowUp, Paperclip, X } from "lucide-react";
import { nanoid } from "nanoid";
import { useCallback, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { chatclawApi } from "@/api/chatclaw";
import { useModels } from "@/core/models/hooks";
import { splitUnsupportedUploadFiles } from "@/core/uploads";
import { cn } from "@/lib/utils";

const CREATE_PRESET_LABEL = "➕ 创建";
const MAX_START_ATTACHMENTS = 8;

function defaultProjectName() {
  return `未命名项目 · ${new Date().toLocaleString("zh-CN", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })}`;
}

const QUICK_PRESETS: { label: string; text: string }[] = [
  { label: "✨ 小惊喜", text: "给我一个小惊喜：分享一个有趣的知识点或小工具。" },
  { label: "📝 写作", text: "我想写一篇文章，主题是：" },
  { label: "🔬 研究", text: "请帮我调研并总结：" },
  { label: "📦 收集", text: "帮我整理和归纳以下方面的资料：" },
  { label: "🎓 学习", text: "我想系统学习以下内容，请给出学习路径：" },
  { label: CREATE_PRESET_LABEL, text: "我想新建一个项目并开始讨论：" },
];

type CreatePayload = {
  message: string;
  files: File[];
  chatMode?: "pro";
};

/**
 * 全屏「新建项目」起点（替代弹窗）：与首页会话输入类似的落地页，提交后创建项目并进入详情页且带上首条消息。
 */
export function NewProjectStartPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { models } = useModels();
  const [text, setText] = useState("");
  const [pro, setPro] = useState(false);
  const [presetFlash, setPresetFlash] = useState(false);
  const [attached, setAttached] = useState<{ id: string; file: File }[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const modelLabel = useMemo(() => {
    const m = models[0];
    return m?.display_name ?? m?.model ?? m?.name ?? "默认模型";
  }, [models]);

  const create = useMutation({
    mutationFn: async ({ message, files }: CreatePayload) => {
      const proj = await chatclawApi.projects.create({
        name: defaultProjectName(),
        description: "",
        category: "",
      });
      return { proj, message, files };
    },
    onSuccess: ({ proj, message, files }, variables) => {
      void qc.invalidateQueries({ queryKey: ["projects"] });
      navigate(`/p/${proj.id}`, {
        replace: false,
        state: {
          initialPrompt: message.trim(),
          initialFiles: files,
          initialChatMode: variables.chatMode,
        },
      });
    },
  });

  const runCreate = useCallback(
    (payload: CreatePayload) => {
      if (create.isPending) return;
      create.mutate(payload);
    },
    [create],
  );

  const submit = useCallback(() => {
    const t = text.trim();
    const fallback = "（见附件）";
    const message = t || (attached.length > 0 ? fallback : "");
    if ((!message && attached.length === 0) || create.isPending) return;
    runCreate({
      message,
      files: attached.map((a) => a.file),
      chatMode: pro ? "pro" : undefined,
    });
  }, [text, attached, pro, create.isPending, runCreate]);

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  };

  const focusTextareaEnd = useCallback((value: string) => {
    requestAnimationFrame(() => {
      const el = textareaRef.current;
      if (!el) return;
      el.focus();
      const len = value.length;
      el.setSelectionRange(len, len);
    });
  }, []);

  const applyPreset = useCallback(
    (p: (typeof QUICK_PRESETS)[number]) => {
      const wasSame = text.trim() === p.text.trim();
      setText(p.text);
      focusTextareaEnd(p.text);
      setPresetFlash(true);
      window.setTimeout(() => setPresetFlash(false), 450);
      if (wasSame && p.label === CREATE_PRESET_LABEL) {
        queueMicrotask(() => {
          if (create.isPending) return;
          runCreate({
            message: p.text.trim(),
            files: attached.map((a) => a.file),
            chatMode: pro ? "pro" : undefined,
          });
        });
      }
    },
    [text, focusTextareaEnd, create.isPending, attached, pro, runCreate],
  );

  const openFilePicker = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const onFileInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const list = e.target.files;
      e.target.value = "";
      if (!list?.length) return;
      const { accepted, message } = splitUnsupportedUploadFiles(list);
      if (message) {
        toast.error(message);
      }
      if (accepted.length === 0) return;
      setAttached((prev) => {
        const room = MAX_START_ATTACHMENTS - prev.length;
        if (room <= 0) {
          toast.error(`最多添加 ${MAX_START_ATTACHMENTS} 个附件`);
          return prev;
        }
        const slice = accepted.slice(0, room);
        if (accepted.length > slice.length) {
          toast.error(`最多添加 ${MAX_START_ATTACHMENTS} 个附件，已忽略多余文件`);
        }
        return prev.concat(
          slice.map((file) => ({ id: nanoid(), file })),
        );
      });
    },
    [],
  );

  const removeAttachment = useCallback((id: string) => {
    setAttached((prev) => prev.filter((a) => a.id !== id));
  }, []);

  const inputWrap =
    "rounded-2xl border border-outline-variant/20 bg-white shadow-sm focus-within:border-black/25 focus-within:ring-2 focus-within:ring-black/10";

  const canSend =
    (text.trim().length > 0 || attached.length > 0) && !create.isPending;

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={onFileInputChange}
      />
      <div className="flex shrink-0 justify-end px-4 pt-4">
        <Link
          to="/"
          className="text-muted-foreground hover:text-foreground text-xs font-medium transition-colors"
        >
          返回项目列表
        </Link>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-6 md:px-10">
        <div className="mx-auto flex max-w-2xl flex-col items-center text-center">
          <p className="mb-3 text-4xl" aria-hidden>
            👋
          </p>
          <h1 className="text-foreground text-2xl font-bold tracking-tight">你好，欢迎回来！</h1>
          <p className="text-muted-foreground mx-auto mt-4 max-w-xl text-sm leading-relaxed">
            欢迎使用 MetaNote。通过内置和自定义的 Skills，可以帮你搜索网络、分析数据，还能为你生成幻灯片、图片、视频、播客及网页等，几乎可以做任何事情。
          </p>

          <div
            className={cn(
              "mt-10 w-full transition-[box-shadow] duration-300",
              inputWrap,
              presetFlash && "ring-2 ring-black/15 border-black/20",
            )}
          >
            <textarea
              ref={textareaRef}
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={onKeyDown}
              placeholder="今天我能为你做些什么？"
              rows={4}
              disabled={create.isPending}
              className="text-foreground placeholder:text-muted-foreground/55 w-full resize-none rounded-2xl bg-transparent px-4 py-3 text-[15px] leading-relaxed outline-none disabled:opacity-60"
            />
            {attached.length > 0 ? (
              <div className="flex flex-wrap gap-1.5 border-t border-black/[0.06] px-4 py-2">
                {attached.map(({ id, file }) => (
                  <span
                    key={id}
                    className="border-border/40 bg-black/[0.03] text-foreground inline-flex max-w-[min(100%,220px)] items-center gap-1 rounded-lg border px-2 py-1 text-xs"
                  >
                    <span className="truncate" title={file.name}>
                      {file.name}
                    </span>
                    <button
                      type="button"
                      disabled={create.isPending}
                      onClick={() => removeAttachment(id)}
                      className="text-muted-foreground hover:text-foreground shrink-0 rounded p-0.5 transition-colors disabled:opacity-40"
                      aria-label={`移除 ${file.name}`}
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </span>
                ))}
              </div>
            ) : null}
            <div className="flex items-center justify-between gap-2 border-t border-black/[0.06] px-3 py-2.5">
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  title="添加附件（将随首条消息一起发送）"
                  onClick={openFilePicker}
                  disabled={create.isPending || attached.length >= MAX_START_ATTACHMENTS}
                  className="text-muted-foreground hover:text-foreground rounded-lg p-2 transition-colors hover:bg-black/[0.04] disabled:pointer-events-none disabled:opacity-35"
                  aria-label="添加附件"
                >
                  <Paperclip className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  title="开启后首条对话使用 Pro 模式（规划与执行）"
                  onClick={() => setPro((v) => !v)}
                  className={cn(
                    "rounded-full px-3 py-1 text-xs font-semibold transition-colors",
                    pro
                      ? "bg-foreground text-background"
                      : "bg-black/[0.06] text-muted-foreground hover:bg-black/[0.08]",
                  )}
                >
                  Pro
                </button>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground hidden max-w-[140px] truncate text-[11px] sm:inline">
                  {modelLabel}
                </span>
                <button
                  type="button"
                  onClick={submit}
                  disabled={!canSend}
                  title="发送"
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-foreground text-background transition-opacity hover:opacity-90 disabled:pointer-events-none disabled:opacity-35"
                >
                  {create.isPending ? (
                    <span className="h-4 w-4 animate-pulse rounded-full bg-background/80" />
                  ) : (
                    <ArrowUp className="h-4 w-4" strokeWidth={2.5} />
                  )}
                </button>
              </div>
            </div>
          </div>

          <div className="mt-6 flex w-full flex-wrap justify-center gap-2">
            {QUICK_PRESETS.map((p) => (
              <button
                key={p.label}
                type="button"
                onClick={() => applyPreset(p)}
                disabled={create.isPending}
                className="border-border/50 text-muted-foreground hover:text-foreground hover:border-foreground/15 rounded-full border bg-white/80 px-3 py-1.5 text-xs font-medium shadow-sm transition-colors disabled:opacity-50"
              >
                {p.label}
              </button>
            ))}
          </div>

          {create.isError ? (
            <p className="text-destructive mt-6 max-w-md text-sm">
              创建项目失败，请检查网络或稍后重试。
            </p>
          ) : null}
        </div>
      </div>
    </div>
  );
}
