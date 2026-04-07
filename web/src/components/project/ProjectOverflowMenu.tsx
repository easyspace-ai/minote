"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Archive, MoreHorizontal, Pencil, Star, Trash2 } from "lucide-react";
import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";

import type { Project } from "@/api/chatclaw";
import { chatclawApi } from "@/api/chatclaw";
import { ProjectEditDialog } from "@/components/project/ProjectEditDialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

type Props = {
  project: Project;
  /** 更小的触发器（例如项目页顶栏） */
  compact?: boolean;
  className?: string;
};

export function ProjectOverflowMenu({ project, compact, className }: Props) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [editOpen, setEditOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const starred = Boolean(project.starred);
  const archived = Boolean(project.archived);

  const patch = useMutation({
    mutationFn: (body: Parameters<typeof chatclawApi.projects.patch>[1]) =>
      chatclawApi.projects.patch(project.id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["projects"] });
      void qc.invalidateQueries({ queryKey: ["project", project.id] });
    },
    onError: (e: Error) => toast.error(e.message || "操作失败"),
  });

  const remove = useMutation({
    mutationFn: () => chatclawApi.projects.remove(project.id),
    onSuccess: () => {
      toast.success("项目已删除");
      void qc.invalidateQueries({ queryKey: ["projects"] });
      void qc.invalidateQueries({ queryKey: ["project", project.id] });
      if (location.pathname.startsWith(`/p/${project.id}`)) {
        navigate("/");
      }
    },
    onError: (e: Error) => toast.error(e.message || "删除失败"),
  });

  const onDelete = () => {
    if (!window.confirm(`确定删除项目「${project.name}」？此操作不可恢复。`)) return;
    remove.mutate();
  };

  return (
    <>
      <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className={cn(
              "text-muted-foreground hover:text-foreground size-8 shrink-0 rounded-lg hover:bg-[#E0E0E0]",
              compact && "size-7",
              className,
            )}
            aria-label="项目菜单"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-[10rem] rounded-xl border-black/[0.08] p-1 shadow-lg">
          <DropdownMenuItem
            className="cursor-pointer rounded-lg"
            onClick={() => {
              setMenuOpen(false);
              setEditOpen(true);
            }}
          >
            <Pencil className="mr-2 h-4 w-4" />
            编辑
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer rounded-lg"
            disabled={patch.isPending}
            onClick={() => {
              patch.mutate({ starred: !starred });
              setMenuOpen(false);
            }}
          >
            <Star className={cn("mr-2 h-4 w-4", starred && "fill-current")} />
            {starred ? "取消收藏" : "收藏"}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer rounded-lg"
            disabled={patch.isPending}
            onClick={() => {
              patch.mutate({ archived: !archived });
              setMenuOpen(false);
              if (!archived && location.pathname.startsWith(`/p/${project.id}`)) {
                navigate("/");
              }
            }}
          >
            <Archive className="mr-2 h-4 w-4" />
            {archived ? "取消归档" : "归档"}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-destructive focus:text-destructive cursor-pointer rounded-lg"
            disabled={remove.isPending}
            onClick={() => {
              setMenuOpen(false);
              onDelete();
            }}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            删除
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <ProjectEditDialog project={project} open={editOpen} onOpenChange={setEditOpen} />
    </>
  );
}
