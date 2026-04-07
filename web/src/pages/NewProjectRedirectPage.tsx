import { useEffect } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";

import type { AppShellOutletContext } from "@/layout/app-shell-outlet-context";

/** 兼容旧书签 `/new`：打开新建弹窗并进入项目总览 */
export function NewProjectRedirectPage() {
  const navigate = useNavigate();
  const { openNewProject } = useOutletContext<AppShellOutletContext>();

  useEffect(() => {
    openNewProject();
    navigate("/projects", { replace: true });
  }, [navigate, openNewProject]);

  return (
    <div className="text-muted-foreground flex h-full items-center justify-center text-sm">正在跳转…</div>
  );
}
