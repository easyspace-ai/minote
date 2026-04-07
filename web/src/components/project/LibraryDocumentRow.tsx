import { FileText } from "lucide-react";
import {
  isLibraryDocumentProcessing,
  libraryDocumentCombinedProgress,
  libraryDocumentStatusParts,
  type LibraryDocumentRow,
} from "@/lib/libraryDocumentStatus";
import { cn } from "@/lib/utils";

type Props = {
  doc: LibraryDocumentRow;
  className?: string;
};

export function LibraryDocumentRow({ doc, className }: Props) {
  const busy = isLibraryDocumentProcessing(doc);
  const pct = libraryDocumentCombinedProgress(doc);
  const { line, detail } = libraryDocumentStatusParts(doc);
  const chunks = doc.split_total ?? 0;
  const sizeKb = (doc.file_size / 1024).toFixed(1);

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-sm dark:shadow-none dark:ring-1 dark:ring-white/10",
        className,
      )}
    >
      <div
        className={cn(
          "absolute bottom-0 left-0 top-0 w-1",
          busy ? "bg-primary" : "bg-muted-foreground/25",
        )}
        aria-hidden
      />
      <div className="pl-4 pr-3 py-2.5">
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-1 items-start gap-2">
            <FileText className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold leading-snug text-foreground">{doc.original_name}</div>
              <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[10px] text-muted-foreground">
                <span>{sizeKb} KB</span>
                <span className="rounded-md border border-border px-1.5 py-0.5 tabular-nums text-muted-foreground">
                  {chunks} 块
                </span>
              </div>
            </div>
          </div>
          <span className="shrink-0 text-[9px] font-semibold uppercase tracking-wide text-muted-foreground">
            FILE
          </span>
        </div>

        {busy ? (
          <div className="mt-2 space-y-1.5">
            <div
              className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
              role="progressbar"
              aria-valuenow={Math.round(pct)}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                className="h-full rounded-full bg-primary transition-[width] duration-300 ease-out"
                style={{ width: `${pct}%` }}
              />
            </div>
            <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground">
              <span>{line}</span>
              <span className="tabular-nums">{Math.round(pct)}%</span>
            </div>
          </div>
        ) : detail ? (
          <p className="mt-2 text-[10px] text-destructive">{detail}</p>
        ) : null}
      </div>
    </div>
  );
}
