/**
 * NotebookLM-style 三栏工作台：顶栏 + 可拖拽宽度 + 左右侧栏动画。
 * 内容通过 left / center / right 注入。
 */
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowUp,
  MoreHorizontal,
  PanelLeftClose,
  PanelLeftOpen,
  PanelRightClose,
  PanelRightOpen,
  Paperclip,
  Search,
  Sparkles,
} from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { Link } from "react-router-dom";

export const NOTEBOOK_WORKBENCH_TABS = ["资料", "对话"] as const;
export type NotebookWorkbenchTab = (typeof NOTEBOOK_WORKBENCH_TABS)[number];

type NotebookShellLayoutValue = {
  closeRightPanel: () => void;
};

const NotebookShellLayoutContext = createContext<NotebookShellLayoutValue | null>(null);

/** 供右侧 Studio 等子组件调用，收起右栏（仅当渲染在 `NotebookShell` 内时有效）。 */
export function useNotebookShellLayout() {
  return useContext(NotebookShellLayoutContext);
}

type Props = {
  /** 左栏固定内容（与 `leftByTab` 二选一）。 */
  left?: React.ReactNode;
  /** 左栏随顶栏「资料 / 对话 / 结果」切换（与 `left` 二选一）。 */
  leftByTab?: Record<NotebookWorkbenchTab, React.ReactNode>;
  /** Default center when `centerByTab` is not used. */
  center?: React.ReactNode;
  /** When set, middle column switches with top tabs (optional; 项目页仅用 `center` 固定会话区). */
  centerByTab?: Record<NotebookWorkbenchTab, React.ReactNode>;
  right: React.ReactNode;
  /** 若传入（如 `/`），顶栏 Logo 可点击返回工作区 */
  logoHref?: string;
  /** Initial tab when using `centerByTab` (default: 对话). */
  defaultWorkbenchTab?: NotebookWorkbenchTab;
  /** 受控：当前顶栏标签（需与 `onWorkbenchTabChange` 同用）。 */
  workbenchTab?: NotebookWorkbenchTab;
  onWorkbenchTabChange?: (tab: NotebookWorkbenchTab) => void;
};

export function NotebookShell({
  left,
  leftByTab,
  center,
  centerByTab,
  right,
  logoHref,
  defaultWorkbenchTab = "对话",
  workbenchTab: workbenchTabControlled,
  onWorkbenchTabChange,
}: Props) {
  const [isLeftOpen, setIsLeftOpen] = useState(true);
  const [isRightOpen, setIsRightOpen] = useState(true);
  const [leftWidth, setLeftWidth] = useState(280);
  const [rightWidth, setRightWidth] = useState(360);
  const isResizingLeft = useRef(false);
  const isResizingRight = useRef(false);

  const startResizingLeft = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isResizingLeft.current = true;
    document.body.style.cursor = "col-resize";
  }, []);

  const startResizingRight = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRight.current = true;
    document.body.style.cursor = "col-resize";
  }, []);

  const stopResizing = useCallback(() => {
    isResizingLeft.current = false;
    isResizingRight.current = false;
    document.body.style.cursor = "default";
  }, []);

  const resize = useCallback((e: MouseEvent) => {
    if (isResizingLeft.current) {
      const nw = e.clientX;
      if (nw > 150 && nw < 500) setLeftWidth(nw);
    } else if (isResizingRight.current) {
      const nw = window.innerWidth - e.clientX;
      if (nw > 220 && nw < 640) setRightWidth(nw);
    }
  }, []);

  useEffect(() => {
    window.addEventListener("mousemove", resize);
    window.addEventListener("mouseup", stopResizing);
    return () => {
      window.removeEventListener("mousemove", resize);
      window.removeEventListener("mouseup", stopResizing);
    };
  }, [resize, stopResizing]);

  const [internalTab, setInternalTab] = useState<NotebookWorkbenchTab>(defaultWorkbenchTab);
  const tabs = NOTEBOOK_WORKBENCH_TABS;

  const isWorkbenchControlled =
    workbenchTabControlled !== undefined && onWorkbenchTabChange !== undefined;
  const activeTab = isWorkbenchControlled ? workbenchTabControlled : internalTab;

  const setActiveTab = useCallback(
    (tab: NotebookWorkbenchTab) => {
      if (isWorkbenchControlled) {
        onWorkbenchTabChange(tab);
      } else {
        setInternalTab(tab);
      }
    },
    [isWorkbenchControlled, onWorkbenchTabChange],
  );

  const resolvedCenter =
    centerByTab != null ? centerByTab[activeTab] : (center ?? null);

  const resolvedLeft =
    leftByTab != null ? leftByTab[activeTab] : (left ?? null);

  const closeRightPanel = useCallback(() => setIsRightOpen(false), []);

  const layoutValue = useMemo<NotebookShellLayoutValue>(
    () => ({ closeRightPanel }),
    [closeRightPanel],
  );

  return (
    <NotebookShellLayoutContext.Provider value={layoutValue}>
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-white text-foreground selection:bg-foreground/10 selection:text-foreground">
      <header className="h-14 flex shrink-0 items-center justify-between border-b border-black/[0.06] bg-white/95 px-4 backdrop-blur-md z-40">
        <div className="flex min-w-0 items-center gap-3">
          {logoHref != null && logoHref !== "" ? (
            <Link
              to={logoHref}
              className="mr-2 flex shrink-0 items-center gap-2 rounded-lg outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-foreground/20"
              title="MetaNote · 返回工作区"
            >
              <img
                src="/logo.jpg"
                alt=""
                className="h-8 w-8 shrink-0 rounded-full object-cover ring-1 ring-black/[0.06]"
              />
              <span className="hidden font-semibold tracking-tight text-foreground sm:inline">MetaNote</span>
            </Link>
          ) : (
            <div className="mr-2 flex shrink-0 items-center gap-2">
              <img
                src="/logo.jpg"
                alt=""
                className="h-8 w-8 shrink-0 rounded-full object-cover ring-1 ring-black/[0.06]"
              />
              <span className="hidden font-semibold tracking-tight text-foreground sm:inline">MetaNote</span>
            </div>
          )}
          <button
            type="button"
            onClick={() => setIsLeftOpen(!isLeftOpen)}
            className="text-muted-foreground hover:text-foreground shrink-0 rounded-full p-1.5 transition-colors hover:bg-[#E0E0E0]"
            title={isLeftOpen ? "收起左栏" : "展开左栏"}
          >
            {isLeftOpen ? <PanelLeftClose className="w-5 h-5" /> : <PanelLeftOpen className="w-5 h-5" />}
          </button>
          <div className="border-border/50 mx-1 h-4 w-px shrink-0 bg-border/60" />
          <nav className="flex items-center gap-4 md:gap-6 shrink-0">
            {tabs.map((tab) => (
              <button
                key={tab}
                type="button"
                onClick={() => setActiveTab(tab as NotebookWorkbenchTab)}
                className={`relative pb-1 text-sm tracking-wide transition-all whitespace-nowrap ${
                  activeTab === tab
                    ? "font-bold text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {tab}
                {activeTab === tab && (
                  <motion.div layoutId="nbTab" className="bg-foreground absolute right-0 bottom-0 left-0 h-0.5" />
                )}
              </button>
            ))}
          </nav>
          {/* <div className="relative ml-2 hidden lg:block w-48 xl:w-64 min-w-0">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground w-3.5 h-3.5" />
            <input
              className="w-full rounded-full border border-black/[0.08] bg-white py-1 pr-4 pl-9 text-xs placeholder:text-muted-foreground/50 focus:border-black/15 focus:ring-0"
              placeholder="搜索工作区…"
              type="search"
            />
          </div> */}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            type="button"
            onClick={() => setIsRightOpen(!isRightOpen)}
            className="text-muted-foreground hover:text-foreground rounded-full p-1.5 transition-colors hover:bg-[#E0E0E0]"
            title={isRightOpen ? "收起右栏" : "展开右栏"}
          >
            {isRightOpen ? <PanelRightClose className="w-5 h-5" /> : <PanelRightOpen className="w-5 h-5" />}
          </button>
        </div>
      </header>

      <div className="relative flex min-h-0 flex-1 gap-2 overflow-hidden p-2">
        <AnimatePresence initial={false}>
          {isLeftOpen && (
            <motion.section
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: leftWidth, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ duration: 0.25, ease: "easeInOut" }}
              className="flex min-h-0 shrink-0 flex-col overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm"
            >
              <div className="flex h-full min-h-0 min-w-[180px] flex-col overflow-hidden p-4 md:p-6">
                {resolvedLeft}
              </div>
            </motion.section>
          )}
        </AnimatePresence>

        {isLeftOpen && (
          <div
            role="separator"
            aria-orientation="vertical"
            onMouseDown={startResizingLeft}
            className="group z-10 flex w-1 shrink-0 cursor-col-resize items-center justify-center transition-colors hover:bg-foreground/10"
          >
            <div className="h-8 w-px bg-border/50 group-hover:bg-foreground/25" />
          </div>
        )}

        <section className="relative flex min-w-0 min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm">
          {resolvedCenter}
        </section>

        {isRightOpen && (
          <div
            role="separator"
            aria-orientation="vertical"
            onMouseDown={startResizingRight}
            className="group z-10 flex w-1 shrink-0 cursor-col-resize items-center justify-center transition-colors hover:bg-foreground/10"
          >
            <div className="h-8 w-px bg-border/50 group-hover:bg-foreground/25" />
          </div>
        )}

        <AnimatePresence initial={false}>
          {isRightOpen && (
            <motion.section
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: rightWidth, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ duration: 0.25, ease: "easeInOut" }}
              className="flex min-h-0 shrink-0 flex-col overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm"
            >
              <div className="flex h-full min-h-0 min-w-[220px] flex-col overflow-y-auto p-3 md:p-4 no-scrollbar">
                {right}
              </div>
            </motion.section>
          )}
        </AnimatePresence>
      </div>
    </div>
    </NotebookShellLayoutContext.Provider>
  );
}

/** 中间输入条：webui 大圆角风格 */
export function ChatComposerBar(props: {
  value: string;
  onChange: (v: string) => void;
  onSend: () => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  const { value, onChange, onSend, disabled, placeholder } = props;
  return (
    <div className="shrink-0 p-6 pt-2">
      <div className="group mx-auto flex max-w-4xl items-center gap-2 rounded-[1.5rem] border border-black/[0.08] bg-white p-2 shadow-sm transition-all focus-within:border-black/15">
        <button type="button" className="p-3 text-muted-foreground hover:text-black transition-colors">
          <Paperclip className="w-5 h-5" />
        </button>
        <input
          className="flex-1 bg-transparent border-none focus:ring-0 text-black text-base px-2 placeholder:text-muted-foreground/50"
          placeholder={placeholder ?? "Ask anything…"}
          type="text"
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              onSend();
            }
          }}
        />
        <button
          type="button"
          disabled={disabled || !value.trim()}
          onClick={onSend}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#E0E0E0] text-foreground transition-all hover:bg-[#D5D5D5] disabled:opacity-40"
        >
          <ArrowUp className="h-5 w-5" />
        </button>
      </div>
    </div>
  );
}

export function MessageBubble(props: { role: string; children: React.ReactNode }) {
  const isUser = props.role === "user";
  return (
    <div className={`flex gap-4 max-w-3xl ${isUser ? "ml-auto justify-end" : ""}`}>
      {!isUser && (
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#E0E0E0]">
          <Sparkles className="h-4 w-4 text-foreground/80" />
        </div>
      )}
      <div className={`space-y-1 ${isUser ? "text-right" : ""}`}>
        <div
          className={`px-5 py-3 rounded-3xl text-base leading-relaxed ${
            isUser ? "bg-surface-container-low rounded-tr-none" : "text-black/90"
          }`}
        >
          {props.children}
        </div>
        <span className="text-[10px] uppercase text-muted-foreground">{props.role}</span>
      </div>
    </div>
  );
}

export function StreamingIndicator() {
  return (
    <div className="flex gap-4 max-w-3xl">
      <div className="w-8 h-8 rounded-full bg-surface-container-high flex items-center justify-center shrink-0 animate-pulse">
        <MoreHorizontal className="text-muted-foreground w-4 h-4" />
      </div>
      <div className="italic text-muted-foreground text-sm animate-pulse">Generating…</div>
    </div>
  );
}
