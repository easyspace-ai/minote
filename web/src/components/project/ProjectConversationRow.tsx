import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { Conversation } from "@/api/chatclaw";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

type Props = {
  conversation: Conversation;
  active: boolean;
  onSelect: () => void;
  onRename: () => void;
  onDelete: () => void;
};

/** 侧栏会话行：左键选中；右键 / 「⋯」重命名、删除（删除为危险样式） */
export function ProjectConversationRow({ conversation, active, onSelect, onRename, onDelete }: Props) {
  const title = conversation.name || `会话 #${conversation.id}`;

  const rowInner = (
    <div
      className={cn(
        "group flex min-w-0 items-center gap-0.5 rounded-xl px-1 transition-colors",
        active ? "bg-[#E0E0E0]" : "",
      )}
    >
      <button
        type="button"
        onClick={onSelect}
        className={cn(
          "min-w-0 flex-1 truncate rounded-lg px-2 py-2 text-left text-sm transition-colors",
          active ? "font-medium text-foreground" : "text-foreground hover:bg-[#E5E5E5]",
        )}
      >
        {title}
      </button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className={cn(
              "text-muted-foreground shrink-0 rounded-lg p-1.5 transition-colors hover:bg-black/6 hover:text-foreground",
              active ? "opacity-100" : "opacity-0 group-hover:opacity-100 data-[state=open]:opacity-100",
            )}
            aria-label={`「${title}」更多操作`}
            onClick={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="size-4" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-[9rem]">
          <DropdownMenuItem
            onSelect={(e) => {
              e.preventDefault();
              onRename();
            }}
          >
            <Pencil className="size-4" />
            重命名
          </DropdownMenuItem>
          <DropdownMenuItem
            variant="destructive"
            onSelect={(e) => {
              e.preventDefault();
              onDelete();
            }}
          >
            <Trash2 className="size-4" />
            删除
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{rowInner}</ContextMenuTrigger>
      <ContextMenuContent className="min-w-[9rem]">
        <ContextMenuItem
          onSelect={(e) => {
            e.preventDefault();
            onRename();
          }}
        >
          <Pencil className="size-4" />
          重命名
        </ContextMenuItem>
        <ContextMenuItem
          variant="destructive"
          onSelect={(e) => {
            e.preventDefault();
            onDelete();
          }}
        >
          <Trash2 className="size-4" />
          删除
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
