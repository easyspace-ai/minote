"use client";

import { useQuery } from "@tanstack/react-query";
import { Lightbulb, LogOut, Package, Plus, Search, Settings } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";

import { chatclawApi, type Project } from "@/api/chatclaw";
import { useAuth } from "@/auth/AuthContext";
import { NewProjectDialog } from "@/components/project/NewProjectDialog";
import { ProjectOverflowMenu } from "@/components/project/ProjectOverflowMenu";
import type { AppShellOutletContext } from "@/layout/app-shell-outlet-context";
import { getProfileDisplayName } from "@/lib/profileDisplayName";
import { projectTileFromProject } from "@/lib/projectAppearance";
import { cn } from "@/lib/utils";

function navRowClass({ isActive }: { isActive: boolean }) {
  return [
    "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
    isActive
      ? "bg-accent/10 text-accent"
      : "text-muted-foreground hover:bg-secondary/10 hover:text-foreground",
  ].join(" ");
}

function newProjectButtonClass() {
  return "flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-medium text-white bg-accent shadow-md hover:bg-accent/90 transition-all hover:shadow-lg";
}

function AppShellProjectRow({ project: p, pathname }: { project: Project; pathname: string }) {
  const isActive = pathname === `/p/${p.id}` || pathname.startsWith(`/p/${p.id}/`);
  const tile = projectTileFromProject(p);
  const TileIcon = tile.Icon;
  return (
    <li>
      <div
        className={cn(
          "flex items-center gap-0.5 rounded-xl pr-0.5 transition-colors",
          isActive ? "bg-accent/10" : "hover:bg-secondary/10",
        )}
      >
        <NavLink
          to={`/p/${p.id}`}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-2.5 rounded-xl px-2 py-2 text-sm",
            isActive ? "font-medium text-foreground" : "text-foreground",
          )}
        >
          <span
            className={cn(
              "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-white shadow-sm",
              tile.bgClass,
            )}
            style={tile.tileStyle}
          >
            <TileIcon className="h-4 w-4 opacity-95" strokeWidth={2} />
          </span>
          <span className="min-w-0 flex-1 truncate">{p.name}</span>
        </NavLink>
        <ProjectOverflowMenu project={p} />
      </div>
    </li>
  );
}

/**
 * 登录后主壳：画布与侧栏白底；`/p/:projectId` 不使用此布局。
 */
export function AppShell() {
  const { user, token, logout } = useAuth();
  const location = useLocation();

  const [searchFocusNonce, setSearchFocusNonce] = useState(0);
  const [newProjectOpen, setNewProjectOpen] = useState(false);
  const [userHovered, setUserHovered] = useState(false);
  const [profileTick, setProfileTick] = useState(0);

  useEffect(() => {
    const onProfile = () => setProfileTick((n) => n + 1);
    window.addEventListener("youmind-profile-updated", onProfile);
    return () => window.removeEventListener("youmind-profile-updated", onProfile);
  }, []);

  const openNewProject = useCallback(() => setNewProjectOpen(true), []);
  const bumpSearchFocus = useCallback(() => setSearchFocusNonce((n) => n + 1), []);

  const outletContext = useMemo<AppShellOutletContext>(
    () => ({ openNewProject, searchFocusNonce }),
    [openNewProject, searchFocusNonce],
  );

  const projects = useQuery({
    queryKey: ["projects"],
    queryFn: () => chatclawApi.projects.list(),
  });

  const sortedProjects = useMemo(() => {
    const list = projects.data ?? [];
    return [...list].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
    );
  }, [projects.data]);

  /** 侧栏不展示已归档项目 */
  const activeProjects = useMemo(
    () => sortedProjects.filter((p) => !p.archived),
    [sortedProjects],
  );
  const favoriteProjects = useMemo(() => activeProjects.filter((p) => p.starred), [activeProjects]);
  const recentSidebarProjects = useMemo(() => activeProjects.filter((p) => !p.starred), [activeProjects]);

  const sidebarDisplayName = useMemo(() => {
    if (!user) return "";
    void profileTick;
    return user.username?.trim() || getProfileDisplayName(user.id, user.email);
  }, [user, profileTick]);

  return (
    <div className="flex h-dvh w-full min-h-0 overflow-hidden bg-background text-foreground">
      <aside className="flex w-[252px] shrink-0 flex-col border-r border-border bg-sidebar">
        <div className="flex h-[52px] shrink-0 items-center justify-between gap-2 border-b border-sidebar-border px-4">
          <Link to="/" className="flex min-w-0 flex-1 items-center gap-3" title="MetaNote">
            <img
              src="/logo.jpg"
              alt=""
              className="h-10 w-10 shrink-0 rounded-full bg-white object-cover ring-1 ring-black/[0.06]"
            />
            <span className="truncate text-lg font-bold tracking-tight text-foreground">MetaNote</span>
          </Link>
          {!user || !token ? (
            <Link
              to="/login"
              className="text-muted-foreground hover:text-foreground shrink-0 text-[11px] font-medium"
            >
              登录
            </Link>
          ) : null}
        </div>

        <div className="flex min-h-0 flex-1 flex-col px-3 pt-3 pb-2">
          <div className="flex shrink-0 flex-col gap-0.5">
            <button type="button" onClick={() => openNewProject()} className={newProjectButtonClass()}>
              <Plus className="h-4 w-4 shrink-0 opacity-90" />
              <span>新建项目</span>
            </button>
            <NavLink
              to="/"
              className={navRowClass}
              end
              onClick={() => bumpSearchFocus()}
            >
              <Search className="h-4 w-4 shrink-0 opacity-90" />
              <span>搜索</span>
            </NavLink>
            <NavLink to="/projects" className={navRowClass}>
              <Package className="h-4 w-4 shrink-0 opacity-90" />
              <span>项目</span>
            </NavLink>
            <NavLink to="/skills" className={navRowClass}>
              <Lightbulb className="h-4 w-4 shrink-0 opacity-90" />
              <span>技能</span>
            </NavLink>
          </div>

          <div className="border-sidebar-border/40 mt-4 flex min-h-0 flex-1 flex-col border-t pt-4">
            <div className="min-h-0 flex-1 overflow-y-auto pr-1 scrollbar-stable">
              {projects.isLoading && (
                <p className="text-muted-foreground px-2 py-2 text-xs">加载中…</p>
              )}
              {projects.error && (
                <p className="text-destructive px-2 py-2 text-xs">项目列表加载失败</p>
              )}
              {favoriteProjects.length > 0 ? (
                <>
                  <p className="text-muted-foreground shrink-0 px-2 text-[11px] font-semibold tracking-wide uppercase">
                    收藏
                  </p>
                  <ul className="mt-2 space-y-0.5 pb-3">
                    {favoriteProjects.map((p: Project) => (
                      <AppShellProjectRow key={p.id} project={p} pathname={location.pathname} />
                    ))}
                  </ul>
                </>
              ) : null}
              <p className="text-muted-foreground shrink-0 px-2 text-[11px] font-semibold tracking-wide uppercase">
                近期项目
              </p>
              <ul className="mt-2 space-y-0.5 pb-2">
                {recentSidebarProjects.map((p: Project) => (
                  <AppShellProjectRow key={p.id} project={p} pathname={location.pathname} />
                ))}
              </ul>
              {!projects.isLoading && !projects.error && activeProjects.length === 0 && (
                <p className="text-muted-foreground px-2 py-2 text-xs">暂无项目，点「新建项目」填写信息即可创建。</p>
              )}
            </div>
          </div>
        </div>

        <div className="shrink-0 border-t border-sidebar-border p-3">
          {user && token ? (
            <div
              className="group flex items-center gap-3 rounded-xl px-2 py-2 transition-colors duration-150 hover:bg-sidebar-accent"
              onMouseEnter={() => setUserHovered(true)}
              onMouseLeave={() => setUserHovered(false)}
            >
              <Link
                to="/settings"
                className="flex min-w-0 flex-1 items-center gap-3 rounded-lg outline-none"
              >
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-gray-200/80 bg-gray-100">
                  <span className="text-sm font-semibold text-gray-600">
                    {(sidebarDisplayName || user.email || "U").charAt(0).toUpperCase()}
                  </span>
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground leading-tight">
                    {sidebarDisplayName || "用户"}
                  </p>
                  {user.credits_balance != null ? (
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      积分: {Math.floor(user.credits_balance)}
                    </p>
                  ) : null}
                </div>
              </Link>
              <div
                className={cn(
                  "flex shrink-0 items-center gap-0.5 transition-opacity duration-150",
                  userHovered ? "opacity-100" : "opacity-0",
                )}
              >
                <Link
                  to="/settings"
                  className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-black/5 dark:hover:bg-white/5 hover:text-foreground"
                  title="设置"
                >
                  <Settings className="h-[15px] w-[15px]" />
                </Link>
                <button
                  type="button"
                  onClick={() => logout()}
                  className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-red-500/10 hover:text-red-600"
                  title="退出"
                >
                  <LogOut className="h-[15px] w-[15px]" />
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </aside>

      <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-tl-[1.5rem] bg-white shadow-md">
        <Outlet context={outletContext} />
      </main>

      <NewProjectDialog open={newProjectOpen} onOpenChange={setNewProjectOpen} />
    </div>
  );
}
