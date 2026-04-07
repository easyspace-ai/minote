import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useOutletContext } from "react-router-dom";

import { chatclawApi, type Project } from "@/api/chatclaw";
import { ProjectOverflowMenu } from "@/components/project/ProjectOverflowMenu";
import type { AppShellOutletContext } from "@/layout/app-shell-outlet-context";
import { projectTileFromProject } from "@/lib/projectAppearance";
import { cn } from "@/lib/utils";

export function WorkspacePage() {
  const { openNewProject, searchFocusNonce } = useOutletContext<AppShellOutletContext>();
  const searchRef = useRef<HTMLInputElement>(null);
  const [q, setQ] = useState("");
  const [showArchived, setShowArchived] = useState(false);

  const projects = useQuery({
    queryKey: ["projects"],
    queryFn: () => chatclawApi.projects.list(),
  });

  useEffect(() => {
    if (searchFocusNonce > 0) {
      searchRef.current?.focus();
    }
  }, [searchFocusNonce]);

  const sorted = useMemo(() => {
    const list = projects.data ?? [];
    return [...list].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
    );
  }, [projects.data]);

  const visible = useMemo(() => {
    if (showArchived) return sorted;
    return sorted.filter((p) => !p.archived);
  }, [sorted, showArchived]);

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return visible;
    return visible.filter(
      (p) =>
        p.name.toLowerCase().includes(s) ||
        (p.description && p.description.toLowerCase().includes(s)) ||
        (p.category && p.category.toLowerCase().includes(s)),
    );
  }, [visible, q]);

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-10 md:px-12">
        <div className="mx-auto max-w-xl">
          <div className="mb-10 text-center">
            {/* <p className="mb-3 text-4xl" aria-hidden>
              🚀
            </p> */}
            <h1 className="text-foreground text-2xl font-bold tracking-tight">你好，欢迎回来！</h1>
            <p className="text-muted-foreground mx-auto mt-3 max-w-md text-sm leading-relaxed">
              MetaNote 助你联网检索、分析数据，并生成幻灯片、网页等多种产物。
            </p>
          </div>

          <div className="relative mb-6">
            <Search className="text-muted-foreground pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2" />
            <input
              ref={searchRef}
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="搜索项目名称…"
              className="border-border/50 text-foreground placeholder:text-muted-foreground/60 focus:ring-foreground/10 w-full rounded-full border bg-white py-3 pr-4 pl-11 text-sm shadow-sm focus:ring-2 focus:outline-none"
            />
          </div>

          <div>
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <h2 className="text-foreground text-sm font-semibold">最近的项目</h2>
              <label className="text-muted-foreground flex cursor-pointer items-center gap-2 text-xs">
                <input
                  type="checkbox"
                  checked={showArchived}
                  onChange={(e) => setShowArchived(e.target.checked)}
                  className="rounded border-black/20"
                />
                显示已归档
              </label>
            </div>
            <div className="max-h-[min(52vh,480px)] overflow-y-auto rounded-2xl border border-black/[0.06] bg-white p-2 shadow-sm scrollbar-stable">
              {projects.isLoading && (
                <p className="text-muted-foreground px-3 py-4 text-sm">加载中…</p>
              )}
              {projects.error && (
                <p className="text-destructive px-3 py-4 text-sm">无法连接后端，请确认服务已启动。</p>
              )}
              <ul className="space-y-0.5">
                {filtered.map((p: Project) => {
                  const tile = projectTileFromProject(p);
                  const TileIcon = tile.Icon;
                  return (
                    <li key={p.id}>
                      <div
                        className={cn(
                          "flex items-center gap-1 rounded-xl pr-1 transition-colors hover:bg-[#F0F0EE]",
                          p.archived && "opacity-70",
                        )}
                      >
                        <Link
                          to={`/p/${p.id}`}
                          className="flex min-w-0 flex-1 items-center gap-3 rounded-xl px-3 py-2.5 text-sm"
                        >
                          <span
                            className={cn(
                              "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-white shadow-sm",
                              tile.bgClass,
                            )}
                            style={tile.tileStyle}
                          >
                            <TileIcon className="h-[18px] w-[18px] opacity-95" strokeWidth={2} />
                          </span>
                          <span className="text-foreground min-w-0 flex-1 truncate font-medium">
                            {p.name}
                            {p.archived ? (
                              <span className="text-muted-foreground ml-2 text-[10px] font-normal">已归档</span>
                            ) : null}
                          </span>
                        </Link>
                        <ProjectOverflowMenu project={p} />
                      </div>
                    </li>
                  );
                })}
              </ul>
              {!projects.isLoading && !projects.error && visible.length === 0 && (
                <p className="text-muted-foreground px-3 py-6 text-center text-sm">
                  暂无项目。点击侧栏「新建项目」在弹窗中填写信息。
                </p>
              )}
              {!projects.isLoading &&
                !projects.error &&
                visible.length > 0 &&
                filtered.length === 0 && (
                  <p className="text-muted-foreground px-3 py-6 text-center text-sm">无匹配项目。</p>
                )}
            </div>
          </div>

          <button
            type="button"
            onClick={() => openNewProject()}
            className="border-border/50 text-muted-foreground hover:text-foreground hover:border-foreground/20 mt-8 flex w-full items-center justify-center gap-2 rounded-2xl border border-dashed bg-white/50 py-3 text-sm font-medium transition-colors"
          >
            <span className="text-lg leading-none">+</span>
            新建项目
          </button>
        </div>
      </div>
    </div>
  );
}
