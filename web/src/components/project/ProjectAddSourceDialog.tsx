"use client";

import { useMutation } from "@tanstack/react-query";
import { FileText } from "lucide-react";
import { useCallback, useId, useRef, useState } from "react";

import { chatclawApi } from "@/api/chatclaw";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result ?? "");
      const i = s.indexOf("base64,");
      resolve(i >= 0 ? s.slice(i + 7) : s);
    };
    r.onerror = reject;
    r.readAsDataURL(file);
  });
}

function utf8ToBase64Payload(text: string): string {
  const bytes = new TextEncoder().encode(text);
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]!);
  return btoa(bin);
}

type TabKey = "upload" | "paste" | "url";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  libraryId: number | null;
  disabled?: boolean;
  onAdded: () => void;
};

export function ProjectAddSourceDialog({
  open,
  onOpenChange,
  libraryId,
  disabled = false,
  onAdded,
}: Props) {
  const fileInputId = useId();
  const fileRef = useRef<HTMLInputElement>(null);
  const [tab, setTab] = useState<TabKey>("upload");
  const [dragOver, setDragOver] = useState(false);
  const [pasteTitle, setPasteTitle] = useState("");
  const [pasteBody, setPasteBody] = useState("");
  const [urlValue, setUrlValue] = useState("");
  const [urlTitle, setUrlTitle] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const resetForms = useCallback(() => {
    setPasteTitle("");
    setPasteBody("");
    setUrlValue("");
    setUrlTitle("");
    setFormError(null);
    setDragOver(false);
    if (fileRef.current) fileRef.current.value = "";
  }, []);

  const handleClose = useCallback(
    (next: boolean) => {
      if (!next) resetForms();
      onOpenChange(next);
    },
    [onOpenChange, resetForms],
  );

  const uploadMutation = useMutation({
    mutationFn: async (files: File[]) => {
      if (libraryId == null || libraryId <= 0) throw new Error("资料库未就绪");
      const payload = [];
      for (const f of files) {
        const b64 = await fileToBase64(f);
        payload.push({ file_name: f.name, base64_data: b64 });
      }
      await chatclawApi.documents.uploadBrowser(libraryId, payload);
    },
    onSuccess: () => {
      onAdded();
      handleClose(false);
    },
    onError: (e) => {
      setFormError(e instanceof Error ? e.message : "上传失败");
    },
  });

  const pasteMutation = useMutation({
    mutationFn: async () => {
      if (libraryId == null || libraryId <= 0) throw new Error("资料库未就绪");
      const title = pasteTitle.trim() || "笔记";
      const body = pasteBody.trim();
      if (!body) throw new Error("请填写内容");
      const safeName = title.endsWith(".txt") ? title : `${title}.txt`;
      const b64 = utf8ToBase64Payload(body);
      await chatclawApi.documents.uploadBrowser(libraryId, [
        { file_name: safeName, base64_data: b64 },
      ]);
    },
    onSuccess: () => {
      onAdded();
      handleClose(false);
    },
    onError: (e) => {
      setFormError(e instanceof Error ? e.message : "添加失败");
    },
  });

  const urlMutation = useMutation({
    mutationFn: async () => {
      if (libraryId == null || libraryId <= 0) throw new Error("资料库未就绪");
      const u = urlValue.trim();
      if (!u) throw new Error("请填写网址");
      await chatclawApi.documents.importUrl(libraryId, {
        url: u,
        title: urlTitle.trim() || undefined,
      });
    },
    onSuccess: () => {
      onAdded();
      handleClose(false);
    },
    onError: (e) => {
      setFormError(e instanceof Error ? e.message : "添加失败");
    },
  });

  const busy =
    uploadMutation.isPending || pasteMutation.isPending || urlMutation.isPending;

  const onPickFiles = useCallback(
    (list: FileList | null) => {
      if (!list?.length || disabled || libraryId == null) return;
      setFormError(null);
      uploadMutation.mutate(Array.from(list));
    },
    [disabled, libraryId, uploadMutation],
  );

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="gap-0 p-0 sm:max-w-lg" showCloseButton>
        <DialogHeader className="border-border border-b px-6 py-4 text-left">
          <DialogTitle>添加来源</DialogTitle>
        </DialogHeader>

        <div className="px-6 pt-4">
          <Tabs
            value={tab}
            onValueChange={(v) => {
              setTab(v as TabKey);
              setFormError(null);
            }}
            className="w-full"
          >
            <TabsList variant="line" className="mb-4 h-auto w-full justify-start gap-6 bg-transparent p-0">
              <TabsTrigger value="upload" className="rounded-none px-0 pb-2 text-sm">
                上传文件
              </TabsTrigger>
              <TabsTrigger value="paste" className="rounded-none px-0 pb-2 text-sm">
                粘贴文本
              </TabsTrigger>
              <TabsTrigger value="url" className="rounded-none px-0 pb-2 text-sm">
                网址
              </TabsTrigger>
            </TabsList>

            {formError ? (
              <p className="text-destructive mb-3 text-xs">{formError}</p>
            ) : null}

            <TabsContent value="upload" className="mt-0 pb-6">
              <input
                ref={fileRef}
                id={fileInputId}
                type="file"
                multiple
                className="sr-only"
                disabled={disabled || libraryId == null || busy}
                onChange={(e) => onPickFiles(e.target.files)}
              />
              <div
                role="button"
                tabIndex={disabled || libraryId == null ? -1 : 0}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    fileRef.current?.click();
                  }
                }}
                onDragOver={(e) => {
                  e.preventDefault();
                  setDragOver(true);
                }}
                onDragLeave={() => setDragOver(false)}
                onDrop={(e) => {
                  e.preventDefault();
                  setDragOver(false);
                  onPickFiles(e.dataTransfer.files);
                }}
                onClick={() => fileRef.current?.click()}
                className={cn(
                  "flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-10 transition-colors",
                  dragOver ? "border-primary bg-primary/5" : "border-muted-foreground/25 bg-muted/30",
                  (disabled || libraryId == null) && "pointer-events-none opacity-50",
                )}
              >
                <FileText className="text-emerald-600 mb-3 h-10 w-10" strokeWidth={1.25} />
                <p className="text-foreground text-center text-sm font-medium">
                  拖放文件到此处或点击浏览
                </p>
                <p className="text-muted-foreground mt-2 max-w-sm text-center text-xs leading-relaxed">
                  支持 PDF、图片、音频、视频、Office 文档（PDF、DOCX、XLSX、PPTX、TXT、MD 等）
                </p>
                {busy ? <p className="text-muted-foreground mt-3 text-xs">上传中…</p> : null}
              </div>
            </TabsContent>

            <TabsContent value="paste" className="mt-0 space-y-4 pb-2">
              <div className="space-y-1.5">
                <label htmlFor="paste-title" className="text-sm font-medium">
                  名称
                </label>
                <Input
                  id="paste-title"
                  placeholder="笔记标题"
                  value={pasteTitle}
                  onChange={(e) => setPasteTitle(e.target.value)}
                  disabled={busy}
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="paste-body" className="text-sm font-medium">
                  内容
                </label>
                <textarea
                  id="paste-body"
                  placeholder="在此粘贴或输入内容…"
                  value={pasteBody}
                  onChange={(e) => setPasteBody(e.target.value)}
                  disabled={busy}
                  rows={8}
                  className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[160px] w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                />
              </div>
            </TabsContent>

            <TabsContent value="url" className="mt-0 space-y-4 pb-2">
              <div className="space-y-1.5">
                <label htmlFor="url-src" className="text-sm font-medium">
                  网址
                </label>
                <Input
                  id="url-src"
                  type="url"
                  placeholder="https://example.com/article"
                  value={urlValue}
                  onChange={(e) => setUrlValue(e.target.value)}
                  disabled={busy}
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="url-name" className="text-sm font-medium">
                  名称（可选）
                </label>
                <Input
                  id="url-name"
                  placeholder="文章标题"
                  value={urlTitle}
                  onChange={(e) => setUrlTitle(e.target.value)}
                  disabled={busy}
                />
              </div>
            </TabsContent>
          </Tabs>
        </div>

        {(tab === "paste" || tab === "url") && (
          <DialogFooter className="border-border gap-2 border-t px-6 py-4">
            <Button type="button" variant="outline" onClick={() => handleClose(false)} disabled={busy}>
              取消
            </Button>
            {tab === "paste" ? (
              <Button
                type="button"
                className="bg-emerald-600 text-white hover:bg-emerald-700"
                disabled={busy}
                onClick={() => {
                  setFormError(null);
                  pasteMutation.mutate();
                }}
              >
                添加来源
              </Button>
            ) : (
              <Button
                type="button"
                className="bg-emerald-600 text-white hover:bg-emerald-700"
                disabled={busy}
                onClick={() => {
                  setFormError(null);
                  urlMutation.mutate();
                }}
              >
                添加来源
              </Button>
            )}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
