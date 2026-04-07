import {
  BarChart3,
  BookOpen,
  ChevronRight,
  Database,
  FileCode,
  FileText,
  LayoutGrid,
  Loader2,
  Mic,
  MoreVertical,
  Network,
  Presentation,
  Settings2,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import type { StudioMaterial, StudioMaterialKind } from "@/api/chatclaw";
import { useArtifactsOptional } from "@/components/workspace/artifacts";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { formatRelativeZh } from "@/lib/formatTime";
import { cn } from "@/lib/utils";
import { getFileName } from "@/core/utils/files";

type Props = {
  materials: StudioMaterial[];
  materialsLoading?: boolean;
  onSelectMaterial: (m: StudioMaterial) => void;
  onQuickMaterial?: (kind: StudioMaterialKind, title: string) => void;
  /** Scope/options UI opened from the header「配置」dialog. */
  scopeControls?: ReactNode;
  /** Persist scope to server; dialog closes only after the promise resolves. */
  onSaveScopeSettings?: () => void | Promise<unknown>;
  scopeSavePending?: boolean;
  /** Thread artifact paths (from chat); shown below Studio materials. */
  onSelectChatArtifact?: (filepath: string) => void;
  /** Studio conversation readiness for disabling tiles and showing status. */
  studioReady?: boolean;
  studioLoading?: boolean;
  studioError?: string | null;
};

type TileBadge = "BETA" | "Soon";

type StudioTile = {
  key: string;
  label: string;
  icon: LucideIcon;
  cardClass: string;
  badge?: TileBadge;
  disabled?: boolean;
  onClick?: () => void;
};

function studioKindLabel(kind: StudioMaterialKind): string {
  const map: Partial<Record<StudioMaterialKind, string>> = {
    audio: "音频",
    slides: "幻灯片",
    html: "网页",
    mindmap: "思维导图",
    report: "报告",
    infographic: "信息图",
    quiz: "测验",
    data_table: "数据表",
  };
  return map[kind] ?? kind;
}

function chatArtifactListLabel(filepath: string): string {
  if (filepath.startsWith("write-file:")) {
    try {
      const u = new URL(filepath);
      return getFileName(decodeURIComponent(u.pathname));
    } catch {
      return filepath;
    }
  }
  return getFileName(filepath);
}

function kindIcon(kind: StudioMaterialKind): { Icon: LucideIcon; wrap: string } {
  switch (kind) {
    case "audio":
      return { Icon: Mic, wrap: "bg-zinc-200 text-zinc-800" };
    case "slides":
      return { Icon: Presentation, wrap: "bg-neutral-200 text-neutral-800" };
    case "html":
      return { Icon: FileCode, wrap: "bg-stone-200 text-stone-800" };
    case "mindmap":
      return { Icon: Network, wrap: "bg-zinc-300 text-zinc-800" };
    case "report":
      return { Icon: FileText, wrap: "bg-neutral-300 text-neutral-800" };
    case "infographic":
      return { Icon: BarChart3, wrap: "bg-stone-300 text-stone-800" };
    case "quiz":
      return { Icon: BookOpen, wrap: "bg-zinc-200 text-zinc-800" };
    case "data_table":
      return { Icon: LayoutGrid, wrap: "bg-neutral-200 text-neutral-800" };
    default:
      return { Icon: FileText, wrap: "bg-muted text-muted-foreground" };
  }
}

export function StudioPanel(props: Props) {
  const {
    materials: materialsProp,
    materialsLoading,
    onSelectMaterial,
    onQuickMaterial,
    scopeControls,
    onSaveScopeSettings,
    scopeSavePending,
    onSelectChatArtifact,
    studioReady = true,
    studioLoading = false,
    studioError = null,
  } = props;

  const materials = materialsProp ?? [];
  const artifactsCtx = useArtifactsOptional();
  const chatArtifactFiles = artifactsCtx?.artifacts ?? [];

  const [scopeOpen, setScopeOpen] = useState(false);

  const tiles: StudioTile[] = [
    {
      key: "audio",
      label: "音频概述",
      icon: Mic,
      cardClass: "bg-[#ECECEC]/90 hover:bg-[#E5E5E5]",
      badge: "BETA",
      onClick: () => onQuickMaterial?.("audio", "新播客概述"),
    },
    {
      key: "slides",
      label: "幻灯片",
      icon: Presentation,
      cardClass: "bg-[#ECECEC]/90 hover:bg-[#E5E5E5]",
      badge: "BETA",
      onClick: () => onQuickMaterial?.("slides", "新幻灯片"),
    },
    {
      key: "mindmap",
      label: "思维导图",
      icon: Network,
      cardClass: "bg-[#E8E8E8]/90 hover:bg-[#E2E2E2]",
      onClick: () => onQuickMaterial?.("mindmap", "新思维导图"),
    },
    {
      key: "report",
      label: "报告",
      icon: FileText,
      cardClass: "bg-[#EBEBEB]/90 hover:bg-[#E5E5E5]",
      onClick: () => onQuickMaterial?.("report", "新报告"),
    },
    {
      key: "quiz",
      label: "测验",
      icon: BookOpen,
      cardClass: "bg-[#EAEAEA]/90 hover:bg-[#E3E3E3]",
      onClick: () => onQuickMaterial?.("quiz", "新测验"),
    },
    {
      key: "infographic",
      label: "信息图",
      icon: BarChart3,
      cardClass: "bg-[#ECECEC]/90 hover:bg-[#E5E5E5]",
      badge: "BETA",
      onClick: () => onQuickMaterial?.("infographic", "新信息图"),
    },
    {
      key: "data_table",
      label: "数据表",
      icon: Database,
      cardClass: "bg-[#E8E8E8]/90 hover:bg-[#E2E2E2]",
      onClick: () => onQuickMaterial?.("data_table", "新数据表"),
    },
    {
      key: "html",
      label: "网页",
      icon: FileCode,
      cardClass: "bg-[#EBEBEB]/90 hover:bg-[#E5E5E5]",
      onClick: () => onQuickMaterial?.("html", "新网页"),
    },
  ];

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="mb-4 flex shrink-0 items-center justify-between gap-2">
        <h2 className="text-foreground text-base font-semibold tracking-tight">Studio</h2>
        {scopeControls ? (
          <Dialog open={scopeOpen} onOpenChange={setScopeOpen}>
            <DialogTrigger asChild>
              <button
                type="button"
                className="text-muted-foreground hover:text-foreground hover:bg-black/[0.04] flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium transition-colors"
                title="生成范围与配置"
              >
                <Settings2 className="h-4 w-4 shrink-0" strokeWidth={2} />
                配置
              </button>
            </DialogTrigger>
            <DialogContent className="max-h-[min(90vh,640px)] overflow-y-auto sm:max-w-lg" showCloseButton>
              <DialogHeader>
                <DialogTitle>生成范围与自定义</DialogTitle>
              </DialogHeader>
              <div className="text-foreground space-y-3 text-xs leading-relaxed">{scopeControls}</div>
              <DialogFooter className="gap-2 sm:gap-0">
                <Button type="button" variant="outline" onClick={() => setScopeOpen(false)}>
                  取消
                </Button>
                <Button
                  type="button"
                  disabled={scopeSavePending}
                  onClick={async () => {
                    if (!onSaveScopeSettings) {
                      setScopeOpen(false);
                      return;
                    }
                    try {
                      await Promise.resolve(onSaveScopeSettings());
                      setScopeOpen(false);
                    } catch {
                      /* error toast from parent mutation */
                    }
                  }}
                >
                  {scopeSavePending ? "保存中…" : "保存"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>

      {/* Studio session status */}
      {!studioReady && (
        <div className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
          {studioLoading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              <span>Studio 会话初始化中…</span>
            </div>
          ) : studioError ? (
            <div className="flex flex-col gap-1">
              <span className="font-medium">Studio 会话初始化失败</span>
              <span className="text-amber-700/80">{studioError}</span>
            </div>
          ) : (
            <span>Studio 会话未就绪，请稍候…</span>
          )}
        </div>
      )}

      <div className="grid shrink-0 grid-cols-2 gap-1.5">
        {tiles.map((tile) => {
          const disabled = Boolean(tile.disabled) || !studioReady || studioLoading;
          return (
            <button
              key={tile.key}
              type="button"
              disabled={disabled}
              onClick={tile.onClick}
              className={cn(
                "flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-left transition-colors",
                "border border-transparent shadow-none",
                tile.cardClass,
                disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer",
              )}
            >
              <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-black/[0.045] text-foreground/75 ring-1 ring-black/[0.06]">
                <tile.icon className="h-3 w-3" strokeWidth={1.5} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-x-1 gap-y-0">
                  <span className="text-foreground text-[10px] font-semibold leading-tight">{tile.label}</span>
                  {tile.badge === "BETA" ? (
                    <span className="bg-foreground text-background rounded px-1 py-px text-[6px] font-bold tracking-wide uppercase">
                      Beta
                    </span>
                  ) : null}
                  {tile.badge === "Soon" ? (
                    <span className="rounded bg-black/[0.08] px-1 py-px text-[6px] font-semibold tracking-wide text-foreground/70 uppercase">
                      Soon
                    </span>
                  ) : null}
                </div>
              </div>
              <ChevronRight
                className={cn(
                  "text-foreground/35 h-3 w-3 shrink-0 self-center",
                  !disabled && "opacity-70",
                )}
                strokeWidth={1.75}
              />
            </button>
          );
        })}
      </div>

      <div className="border-border/50 my-5 h-px shrink-0 bg-border/60" />

      <ul className="scrollbar-stable min-h-0 flex-1 space-y-1 overflow-y-auto pr-0.5">
        {materialsLoading && (
          <li className="text-muted-foreground flex items-center gap-2 px-2 py-2 text-xs">
            <Loader2 className="h-4 w-4 animate-spin" /> 加载中…
          </li>
        )}

        {!materialsLoading && materials.length === 0 && chatArtifactFiles.length === 0 && (
          <li className="text-muted-foreground px-2 py-4 text-center text-xs leading-relaxed">
            暂无生成项；使用上方工具开始。
          </li>
        )}

        {materials.map((m) => {
          const { Icon, wrap } = kindIcon(m.kind);
          const inFlight = m.status === "pending" || m.status === "processing";
          const meta = [
            m.status === "failed"
              ? "生成失败"
              : inFlight
                ? m.subtitle?.trim() || "生成中…"
                : (m.subtitle?.trim() || studioKindLabel(m.kind)),
            formatRelativeZh(m.updated_at),
          ]
            .filter(Boolean)
            .join(" · ");

          return (
            <li key={m.id}>
              <div className="group flex items-start gap-2 rounded-xl p-1.5 transition-colors hover:bg-black/[0.03]">
                <button
                  type="button"
                  onClick={() => onSelectMaterial(m)}
                  className="flex min-w-0 flex-1 items-start gap-3 rounded-lg py-1 pr-1 text-left"
                >
                  <div className={cn("relative flex h-10 w-10 shrink-0 items-center justify-center rounded-xl", wrap)}>
                    <Icon className="h-[18px] w-[18px]" strokeWidth={2} />
                    {inFlight ? (
                      <Loader2 className="text-foreground absolute -right-1 -bottom-1 h-3.5 w-3.5 animate-spin rounded-full bg-white p-0.5" />
                    ) : null}
                  </div>
                  <div className="min-w-0 flex-1 pt-0.5">
                    <p className="text-foreground truncate text-sm font-semibold leading-snug">{m.title}</p>
                    <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs">{meta}</p>
                  </div>
                </button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      type="button"
                      className="text-muted-foreground hover:text-foreground mt-1 shrink-0 rounded-lg p-1.5 opacity-70 transition-opacity group-hover:opacity-100"
                      aria-label="更多"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <MoreVertical className="h-4 w-4" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-40">
                    <DropdownMenuItem onClick={() => onSelectMaterial(m)}>查看</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </li>
          );
        })}

        {chatArtifactFiles.length > 0 && onSelectChatArtifact ? (
          <>
            <li className="text-muted-foreground px-2 pt-4 pb-1 text-[11px] font-semibold tracking-tight">
              对话生成
            </li>
            {chatArtifactFiles.map((filepath) => (
              <li key={filepath}>
                <button
                  type="button"
                  onClick={() => onSelectChatArtifact(filepath)}
                  className="hover:bg-black/[0.03] flex w-full items-start gap-3 rounded-xl p-2.5 text-left transition-colors"
                >
                  <div
                    className={cn(
                      "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                      "bg-stone-200 text-stone-800",
                    )}
                  >
                    <FileCode className="h-[18px] w-[18px]" strokeWidth={2} />
                  </div>
                  <div className="min-w-0 flex-1 pt-0.5">
                    <p className="text-foreground truncate text-sm font-semibold leading-snug">
                      {chatArtifactListLabel(filepath)}
                    </p>
                    <p className="text-muted-foreground mt-0.5 text-xs">来自当前会话</p>
                  </div>
                </button>
              </li>
            ))}
          </>
        ) : null}
      </ul>

    </div>
  );
}
