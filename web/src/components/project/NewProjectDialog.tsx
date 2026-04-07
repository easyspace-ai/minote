"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { chatclawApi } from "@/api/chatclaw";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function NewProjectDialog({ open, onOpenChange }: Props) {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("");

  useEffect(() => {
    if (!open) return;
    setName("");
    setDescription("");
    setCategory("");
  }, [open]);

  const create = useMutation({
    mutationFn: () =>
      chatclawApi.projects.create({
        name: name.trim(),
        description: description.trim(),
        category: category.trim(),
      }),
    onSuccess: (proj) => {
      void qc.invalidateQueries({ queryKey: ["projects"] });
      onOpenChange(false);
      navigate(`/p/${proj.id}`);
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "创建失败");
    },
  });

  const submit = () => {
    if (!name.trim()) {
      toast.error("请填写项目名称");
      return;
    }
    if (create.isPending) return;
    create.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md rounded-2xl sm:max-w-md" showCloseButton>
        <DialogHeader>
          <DialogTitle>新建项目</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 pt-1">
          <div>
            <label htmlFor="np-name" className="text-muted-foreground mb-1.5 block text-xs font-medium">
              项目名称 <span className="text-destructive">*</span>
            </label>
            <input
              id="np-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：产品调研、读书笔记"
              className="border-input bg-background text-foreground focus-visible:ring-ring w-full rounded-xl border px-3 py-2 text-sm outline-none focus-visible:ring-2"
              autoFocus
            />
          </div>
          <div>
            <label htmlFor="np-desc" className="text-muted-foreground mb-1.5 block text-xs font-medium">
              描述（可选）
            </label>
            <textarea
              id="np-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="简要说明项目目标或范围"
              rows={3}
              className="border-input bg-background text-foreground focus-visible:ring-ring w-full resize-y rounded-xl border px-3 py-2 text-sm outline-none focus-visible:ring-2"
            />
          </div>
          <div>
            <label htmlFor="np-cat" className="text-muted-foreground mb-1.5 block text-xs font-medium">
              分类（可选）
            </label>
            <input
              id="np-cat"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="如 note、research"
              className="border-input bg-background text-foreground focus-visible:ring-ring w-full rounded-xl border px-3 py-2 text-sm outline-none focus-visible:ring-2"
            />
          </div>
        </div>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button type="button" onClick={() => submit()} disabled={create.isPending}>
            {create.isPending ? "创建中…" : "创建并打开"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
