import type { CSSProperties } from "react";
import type { LucideIcon } from "lucide-react";
import {
  Atom,
  BookOpen,
  Box,
  Database,
  Flame,
  Gem,
  Heart,
  Lightbulb,
  MapPin,
  PenLine,
  Sparkles,
  TreePine,
} from "lucide-react";

import type { Project } from "@/api/chatclaw";

/** 与编辑弹窗共用的 12 套图标（顺序与 icon_index 对齐） */
export const PROJECT_ICON_PRESETS: Array<{ Icon: LucideIcon }> = [
  { Icon: MapPin },
  { Icon: BookOpen },
  { Icon: Atom },
  { Icon: Database },
  { Icon: Gem },
  { Icon: Heart },
  { Icon: TreePine },
  { Icon: PenLine },
  { Icon: Box },
  { Icon: Flame },
  { Icon: Lightbulb },
  { Icon: Sparkles },
];

/** 参考稿 12 色圆点 */
export const PROJECT_ACCENT_PALETTE: string[] = [
  "#000000",
  "#6B7280",
  "#2563EB",
  "#0D9488",
  "#16A34A",
  "#9333EA",
  "#C084FC",
  "#EC4899",
  "#EF4444",
  "#EA580C",
  "#EAB308",
  "#92400E",
];

const FALLBACK_BG: string[] = [
  "bg-zinc-600",
  "bg-neutral-600",
  "bg-stone-600",
  "bg-zinc-700",
  "bg-neutral-700",
  "bg-stone-500",
  "bg-zinc-500",
  "bg-neutral-500",
  "bg-stone-700",
  "bg-zinc-800",
  "bg-neutral-800",
  "bg-stone-800",
];

/** Stable small int from project uuid (fallback when icon_index is negative). */
export function hashProjectId(id: string): number {
  let h = 0;
  for (let i = 0; i < id.length; i++) {
    h = (h * 31 + id.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

function resolvedIconIndex(p: Pick<Project, "id" | "icon_index">): number {
  const raw = p.icon_index;
  if (typeof raw === "number" && raw >= 0) {
    return raw % PROJECT_ICON_PRESETS.length;
  }
  return hashProjectId(p.id) % PROJECT_ICON_PRESETS.length;
}

export function projectTileFromProject(
  p: Pick<Project, "id" | "icon_index" | "accent_hex">,
): {
  Icon: LucideIcon;
  bgClass: string;
  tileStyle?: CSSProperties;
} {
  const idx = resolvedIconIndex(p);
  const { Icon } = PROJECT_ICON_PRESETS[idx]!;
  const hex = (p.accent_hex ?? "").trim();
  if (/^#[0-9A-Fa-f]{6}$/.test(hex)) {
    return { Icon, bgClass: "", tileStyle: { backgroundColor: hex } };
  }
  return { Icon, bgClass: FALLBACK_BG[idx]! };
}
