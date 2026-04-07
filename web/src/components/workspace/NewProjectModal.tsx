import { X } from "lucide-react";
import { useState } from "react";

type Props = {
  open: boolean;
  onClose: () => void;
  onSubmit: (v: { name: string; description: string; category: string }) => void;
  busy?: boolean;
};

export function NewProjectModal({ open, onClose, onSubmit, busy }: Props) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("");

  if (!open) return null;

  const input =
    "w-full rounded-xl border border-outline-variant/25 bg-surface-container-low px-3 py-2 text-sm text-on-surface placeholder:text-muted-foreground/50 focus:border-black/30 focus:outline-none focus:ring-2 focus:ring-black/10";

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="max-h-[90vh] w-full max-w-md overflow-hidden rounded-2xl border border-outline-variant/15 bg-white whisper-shadow">
        <div className="flex items-center justify-between border-b border-outline-variant/10 px-5 py-4">
          <h2 className="text-lg font-bold">新建项目</h2>
          <button
            type="button"
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-surface-container-high"
            onClick={onClose}
            aria-label="关闭"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 px-5 py-4">
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            <span>
              名称 <span className="text-red-600">*</span>
            </span>
            <input
              className={input}
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：华东仓配优化"
              maxLength={30}
            />
          </label>
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            <span>分类标签</span>
            <input
              className={input}
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="战略 / 研究 / 运营…"
              maxLength={64}
            />
          </label>
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            <span>描述</span>
            <textarea
              className={`${input} min-h-[88px] resize-y`}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="简要说明项目目标…"
              maxLength={2000}
              rows={3}
            />
          </label>
        </div>
        <div className="flex justify-end gap-2 border-t border-outline-variant/10 px-5 py-4">
          <button
            type="button"
            className="rounded-xl px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-surface-container-high"
            onClick={onClose}
          >
            取消
          </button>
          <button
            type="button"
            className="rounded-xl bg-black px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-40"
            disabled={busy || !name.trim()}
            onClick={() => onSubmit({ name: name.trim(), description: description.trim(), category: category.trim() })}
          >
            {busy ? "创建中…" : "创建"}
          </button>
        </div>
      </div>
    </div>
  );
}
