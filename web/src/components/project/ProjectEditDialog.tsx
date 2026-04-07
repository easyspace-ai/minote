"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

import type { Project } from "@/api/chatclaw";
import { chatclawApi } from "@/api/chatclaw";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  PROJECT_ACCENT_PALETTE,
  PROJECT_ICON_PRESETS,
  projectTileFromProject,
} from "@/lib/projectAppearance";
import { cn } from "@/lib/utils";

function defaultIconIndex(p: Project): number {
  const raw = p.icon_index;
  if (typeof raw === "number" && raw >= 0) {
    return raw % PROJECT_ICON_PRESETS.length;
  }
  return Math.abs(Math.floor(p.id)) % PROJECT_ICON_PRESETS.length;
}

function defaultAccentHex(p: Project): string {
  const h = (p.accent_hex ?? "").trim();
  if (/^#[0-9A-Fa-f]{6}$/.test(h)) return h;
  return PROJECT_ACCENT_PALETTE[defaultIconIndex(p)] ?? PROJECT_ACCENT_PALETTE[0]!;
}

type Props = {
  project: Project;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function ProjectEditDialog({ project, open, onOpenChange }: Props) {
  const qc = useQueryClient();
  const [name, setName] = useState(project.name);
  const [iconIndex, setIconIndex] = useState(() => defaultIconIndex(project));
  const [accentHex, setAccentHex] = useState(() => defaultAccentHex(project));

  useEffect(() => {
    if (!open) return;
    setName(project.name);
    setIconIndex(defaultIconIndex(project));
    setAccentHex(defaultAccentHex(project));
  }, [open, project.id, project.name, project.icon_index, project.accent_hex, project.updated_at]);

  const preview = useMemo(
    () =>
      projectTileFromProject({
        id: project.id,
        icon_index: iconIndex,
        accent_hex: accentHex,
      }),
    [project.id, iconIndex, accentHex],
  );
  const PreviewIcon = preview.Icon;

  const patch = useMutation({
    mutationFn: () =>
      chatclawApi.projects.patch(project.id, {
        name: name.trim(),
        icon_index: iconIndex,
        accent_hex: accentHex,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["projects"] });
      void qc.invalidateQueries({ queryKey: ["project", project.id] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md rounded-2xl sm:max-w-md" showCloseButton>
        <DialogHeader>
          <DialogTitle>编辑项目</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <label className="text-muted-foreground mb-1.5 block text-xs font-medium">名称</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="border-input bg-background text-foreground focus-visible:ring-ring w-full rounded-xl border px-3 py-2 text-sm outline-none focus-visible:ring-2"
            />
          </div>
          <div>
            <p className="text-muted-foreground mb-2 text-xs font-medium">图标</p>
            <div className="grid grid-cols-6 gap-2">
              {PROJECT_ICON_PRESETS.map((preset, i) => {
                const Icon = preset.Icon;
                const sel = i === iconIndex;
                return (
                  <button
                    key={i}
                    type="button"
                    onClick={() => setIconIndex(i)}
                    className={cn(
                      "flex h-10 w-10 items-center justify-center rounded-xl border transition-colors",
                      sel ? "border-foreground ring-2 ring-foreground/20" : "border-transparent bg-[#E8E8E8] hover:bg-[#E0E0E0]",
                    )}
                  >
                    <Icon className="h-4 w-4 text-foreground" strokeWidth={2} />
                  </button>
                );
              })}
            </div>
          </div>
          <div>
            <p className="text-muted-foreground mb-2 text-xs font-medium">颜色</p>
            <div className="grid grid-cols-6 gap-2">
              {PROJECT_ACCENT_PALETTE.map((hex) => {
                const sel = accentHex.toLowerCase() === hex.toLowerCase();
                return (
                  <button
                    key={hex}
                    type="button"
                    title={hex}
                    onClick={() => setAccentHex(hex)}
                    className={cn(
                      "mx-auto flex h-8 w-8 rounded-full border-2 transition-transform hover:scale-105",
                      sel ? "border-foreground scale-105 ring-2 ring-foreground/25" : "border-white shadow-sm",
                    )}
                    style={{ backgroundColor: hex }}
                  />
                );
              })}
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl border border-black/[0.06] bg-white p-3">
            <span
              className={cn(
                "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-white shadow-sm",
                preview.bgClass,
              )}
              style={preview.tileStyle}
            >
              <PreviewIcon className="h-5 w-5 opacity-95" strokeWidth={2} />
            </span>
            <span className="truncate text-sm font-medium text-foreground">{name.trim() || project.name}</span>
          </div>
        </div>
        <DialogFooter className="gap-2 sm:gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            type="button"
            className="rounded-xl bg-foreground text-background hover:bg-foreground/90"
            disabled={!name.trim() || patch.isPending}
            onClick={() => patch.mutate()}
          >
            {patch.isPending ? "保存中…" : "更新"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
