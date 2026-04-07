import { useCallback, useId, useRef, useState } from "react";
import { Upload, FileUp } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type Props = {
  /** When false, dropzone accepts interaction */
  disabled?: boolean;
  /** Shows progress bar */
  isUploading?: boolean;
  title?: string;
  description?: string;
  /** Compact layout for sidebars (e.g. project studio) */
  compact?: boolean;
  error?: React.ReactNode;
  onFiles: (files: File[]) => void;
};

export function DocumentUploadDropzone({
  disabled = false,
  isUploading = false,
  title = "上传文档",
  description = "拖拽文件到此处，或点击下方选择文件。支持多选。",
  compact = false,
  error,
  onFiles,
}: Props) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  const handleFiles = useCallback(
    (list: FileList | null) => {
      if (!list?.length || disabled) return;
      onFiles(Array.from(list));
      if (inputRef.current) {
        inputRef.current.value = "";
      }
    },
    [disabled, onFiles]
  );

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      handleFiles(e.dataTransfer.files);
    },
    [handleFiles]
  );

  return (
    <Card
      className={cn(
        "border-dashed transition-colors",
        dragOver && !disabled && "border-primary bg-primary/5",
        disabled && "opacity-60"
      )}
    >
      <CardHeader className={cn(!compact && "pb-2")}>
        <CardTitle className={cn("flex items-center gap-2 text-base", compact && "text-sm")}>
          <FileUp className="size-4 shrink-0" aria-hidden />
          {title}
        </CardTitle>
        {!compact && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className="space-y-4">
        <div
          role="button"
          tabIndex={disabled ? -1 : 0}
          onKeyDown={(e) => {
            if (disabled) return;
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              inputRef.current?.click();
            }
          }}
          onDragEnter={(e) => {
            e.preventDefault();
            if (!disabled) setDragOver(true);
          }}
          onDragLeave={(e) => {
            e.preventDefault();
            if (!e.currentTarget.contains(e.relatedTarget as Node)) {
              setDragOver(false);
            }
          }}
          onDragOver={(e) => {
            e.preventDefault();
            if (!disabled) setDragOver(true);
          }}
          onDrop={onDrop}
          className={cn(
            "flex min-h-[120px] cursor-pointer flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-muted/30 px-4 py-6 text-center outline-none transition-colors",
            compact && "min-h-[88px] py-4",
            !disabled && "hover:border-primary/50 hover:bg-muted/50",
            disabled && "cursor-not-allowed"
          )}
          onClick={() => !disabled && inputRef.current?.click()}
        >
          <Upload className="size-8 text-muted-foreground" aria-hidden />
          <p className="text-sm text-muted-foreground">
            {compact ? "拖放或点击上传" : "将文件拖放到此处"}
          </p>
          <Button
            type="button"
            size="sm"
            variant="secondary"
            disabled={disabled}
            onClick={(e) => {
              e.stopPropagation();
              inputRef.current?.click();
            }}
          >
            选择文件
          </Button>
          <input
            ref={inputRef}
            id={inputId}
            type="file"
            multiple
            className="sr-only"
            disabled={disabled}
            aria-label="选择要上传的文档"
            onChange={(e) => handleFiles(e.target.files)}
            onClick={(e) => e.stopPropagation()}
          />
        </div>

        {isUploading && (
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">正在上传并处理…</p>
            <div
              className="h-1 w-full overflow-hidden rounded-full bg-muted"
              role="progressbar"
              aria-busy="true"
              aria-label="上传进度"
            >
              <div className="h-full w-1/3 animate-pulse rounded-full bg-primary motion-reduce:animate-none" />
            </div>
          </div>
        )}

        {error != null && error !== false && (
          <Alert className="border-border bg-popover text-popover-foreground shadow-sm dark:shadow-none dark:ring-1 dark:ring-white/10">
            <AlertTitle className="text-foreground">上传失败</AlertTitle>
            <AlertDescription className="text-muted-foreground">{error}</AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  );
}
