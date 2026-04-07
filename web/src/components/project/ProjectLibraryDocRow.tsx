import {
  ArrowRight,
  CalendarPlus,
  Download,
  Eye,
  FilePlus,
  FileText,
  MoreHorizontal,
  Pencil,
  Star,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import {
  isLibraryDocumentProcessing,
  libraryDocumentCombinedProgress,
  libraryDocumentStatusParts,
  type LibraryDocumentRow,
} from "@/lib/libraryDocumentStatus";
import { cn } from "@/lib/utils";
import {
  ContextMenu,
  ContextMenuCheckboxItem,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuSub,
  ContextMenuSubContent,
  ContextMenuSubTrigger,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  doc: LibraryDocumentRow;
  className?: string;
  /** When set, show a checkbox for Studio「生成选用」(click does not propagate). */
  studioSelect?: {
    checked: boolean;
    onChange: (checked: boolean) => void;
  };
  /** Click row (outside checkbox / more) to open preview. */
  onPreview?: () => void;
  /** Download original file (chat-attachment). */
  onDownload?: () => void | Promise<void>;
  /** Open add-material flow (e.g. file picker dialog). */
  onAddMaterial?: () => void;
  /** Delete document after confirm (handled in parent). */
  onDelete?: () => void;
  /** Rename via prompt/dialog in parent → PATCH original_name. */
  onRename?: () => void;
  /** Toggle favorite → PATCH starred. */
  onStarredChange?: (starred: boolean) => void;
};

type MenuPrimitives = {
  Item: typeof ContextMenuItem | typeof DropdownMenuItem;
  Separator: typeof ContextMenuSeparator | typeof DropdownMenuSeparator;
  CheckboxItem: typeof ContextMenuCheckboxItem | typeof DropdownMenuCheckboxItem;
  Sub: typeof ContextMenuSub | typeof DropdownMenuSub;
  SubTrigger: typeof ContextMenuSubTrigger | typeof DropdownMenuSubTrigger;
  SubContent: typeof ContextMenuSubContent | typeof DropdownMenuSubContent;
};

function LibraryDocMenuItems(
  props: MenuPrimitives & {
    doc: LibraryDocumentRow;
    studioSelect?: Props["studioSelect"];
    onPreview?: () => void;
    onDownload?: () => void | Promise<void>;
    onAddMaterial?: () => void;
    onDelete?: () => void;
    onRename?: () => void;
    onStarredChange?: (starred: boolean) => void;
  },
) {
  const {
    Item,
    Separator,
    CheckboxItem,
    Sub,
    SubTrigger,
    SubContent,
    doc,
    studioSelect,
    onPreview,
    onDownload,
    onAddMaterial,
    onDelete,
    onRename,
    onStarredChange,
  } = props;

  return (
    <>
      {onPreview ? (
        <Item
          onSelect={(e) => {
            e.preventDefault();
            onPreview();
          }}
        >
          <Eye className="size-4" />
          预览
        </Item>
      ) : null}
      {onDownload ? (
        <Item
          onSelect={(e) => {
            e.preventDefault();
            void onDownload();
          }}
        >
          <Download className="size-4" />
          下载
        </Item>
      ) : null}
      {onPreview || onDownload ? <Separator /> : null}

      <Item
        onSelect={(e) => {
          e.preventDefault();
          toast.message("即将推出");
        }}
      >
        <CalendarPlus className="size-4" />
        新建笔记
      </Item>
      {onAddMaterial ? (
        <Item
          onSelect={(e) => {
            e.preventDefault();
            onAddMaterial();
          }}
        >
          <FilePlus className="size-4" />
          添加资料
        </Item>
      ) : null}

      <Separator />

      {onRename ? (
        <Item
          onSelect={(e) => {
            e.preventDefault();
            onRename();
          }}
        >
          <Pencil className="size-4" />
          重命名
        </Item>
      ) : null}
      {onStarredChange ? (
        <CheckboxItem
          checked={Boolean(doc.starred)}
          onCheckedChange={(v) => onStarredChange(v === true)}
        >
          收藏
        </CheckboxItem>
      ) : null}
      {studioSelect ? (
        <CheckboxItem
          checked={studioSelect.checked}
          onCheckedChange={(v) => studioSelect.onChange(v === true)}
        >
          Studio 生成选用
        </CheckboxItem>
      ) : null}

      <Sub>
        <SubTrigger>
          <ArrowRight className="size-4" />
          移动到
        </SubTrigger>
        <SubContent className="min-w-[10rem]">
          <Item disabled>暂无其他位置</Item>
        </SubContent>
      </Sub>

      {onDelete ? (
        <>
          <Separator />
          <Item
            variant="destructive"
            onSelect={(e) => {
              e.preventDefault();
              onDelete();
            }}
          >
            <Trash2 className="size-4" />
            移动到回收站
          </Item>
        </>
      ) : null}
    </>
  );
}

/** 侧栏资料列表：图三风格 — 圆角行、图标 + 标题；右键 / 「⋯」打开菜单 */
export function ProjectLibraryDocRow({
  doc,
  className,
  studioSelect,
  onPreview,
  onDownload,
  onAddMaterial,
  onDelete,
  onRename,
  onStarredChange,
}: Props) {
  const busy = isLibraryDocumentProcessing(doc);
  const pct = libraryDocumentCombinedProgress(doc);
  const { detail } = libraryDocumentStatusParts(doc);
  const lower = doc.original_name.toLowerCase();
  const isPdf = lower.endsWith(".pdf");

  const icon = isPdf ? (
    <span
      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-600 text-[10px] font-bold text-white"
      aria-hidden
    >
      PDF
    </span>
  ) : (
    <div className="bg-muted/80 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg">
      <FileText className="text-muted-foreground h-4 w-4" aria-hidden />
    </div>
  );

  const textBlock = (
    <div className="min-w-0 flex-1">
      <p className="flex min-w-0 items-center gap-1 text-sm font-medium text-foreground">
        {doc.starred ? (
          <Star className="size-3.5 shrink-0 fill-amber-400 text-amber-500" aria-label="已收藏" />
        ) : null}
        <span className="truncate">{doc.original_name}</span>
      </p>
      {busy ? (
        <div
          className="mt-1.5 h-0.5 w-full overflow-hidden rounded-full bg-muted"
          role="progressbar"
          aria-valuenow={Math.round(pct)}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className="bg-foreground/80 h-full rounded-full transition-[width] duration-300 ease-out"
            style={{ width: `${pct}%` }}
          />
        </div>
      ) : detail ? (
        <p className="text-destructive mt-1 text-[10px] leading-tight">{detail}</p>
      ) : null}
    </div>
  );

  const menuProps = {
    doc,
    studioSelect,
    onPreview,
    onDownload,
    onAddMaterial,
    onDelete,
    onRename,
    onStarredChange,
  };

  const rowInner = (
    <div
      className={cn(
        "flex min-w-0 items-center gap-1 rounded-xl px-3 py-2 transition-colors",
        onPreview ? "" : "hover:bg-black/4",
        className,
      )}
    >
      {studioSelect ? (
        <label
          className="flex shrink-0 cursor-pointer items-center pt-0.5"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <input
            type="checkbox"
            className="border-border text-foreground h-4 w-4 rounded accent-black"
            checked={studioSelect.checked}
            onChange={(e) => studioSelect.onChange(e.target.checked)}
            aria-label={`Studio 生成包含 ${doc.original_name}`}
          />
        </label>
      ) : null}
      {onPreview ? (
        <button
          type="button"
          onClick={onPreview}
          className="hover:bg-black/4 flex min-w-0 flex-1 cursor-pointer items-center gap-2.5 rounded-lg px-0 py-0 text-left outline-offset-2 focus-visible:outline-2 focus-visible:outline-ring"
        >
          {icon}
          {textBlock}
        </button>
      ) : (
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          {icon}
          {textBlock}
        </div>
      )}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="text-muted-foreground hover:bg-black/6 hover:text-foreground shrink-0 rounded-lg p-1.5 transition-colors"
            aria-label={`「${doc.original_name}」更多操作`}
            onClick={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="size-4" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-[11rem]">
          <LibraryDocMenuItems
            Item={DropdownMenuItem}
            Separator={DropdownMenuSeparator}
            CheckboxItem={DropdownMenuCheckboxItem}
            Sub={DropdownMenuSub}
            SubTrigger={DropdownMenuSubTrigger}
            SubContent={DropdownMenuSubContent}
            {...menuProps}
          />
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{rowInner}</ContextMenuTrigger>
      <ContextMenuContent className="min-w-[11rem]">
        <LibraryDocMenuItems
          Item={ContextMenuItem}
          Separator={ContextMenuSeparator}
          CheckboxItem={ContextMenuCheckboxItem}
          Sub={ContextMenuSub}
          SubTrigger={ContextMenuSubTrigger}
          SubContent={ContextMenuSubContent}
          {...menuProps}
        />
      </ContextMenuContent>
    </ContextMenu>
  );
}
