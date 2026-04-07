"use client";

import { useCallback, useEffect, useLayoutEffect, useState } from "react";
import { Outlet } from "react-router-dom";

import { PromptInputProvider } from "@/components/ai-elements/prompt-input";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { CommandPalette } from "@/components/workspace/command-palette";
import { WorkspaceSidebar } from "@/components/workspace/workspace-sidebar";
import { getLocalSettings, useLocalSettings } from "@/core/settings";

export function WorkspaceLayout() {
  const [settings, setSettings] = useLocalSettings();
  const [open, setOpen] = useState(false);
  useLayoutEffect(() => {
    setOpen(!getLocalSettings().layout.sidebar_collapsed);
  }, []);
  useEffect(() => {
    setOpen(!settings.layout.sidebar_collapsed);
  }, [settings.layout.sidebar_collapsed]);
  const handleOpenChange = useCallback(
    (next: boolean) => {
      setOpen(next);
      setSettings("layout", { sidebar_collapsed: !next });
    },
    [setSettings],
  );
  return (
    <>
      <SidebarProvider
        className="h-screen"
        open={open}
        onOpenChange={handleOpenChange}
      >
        <WorkspaceSidebar />
        <SidebarInset className="min-w-0">
          <PromptInputProvider>
            <Outlet />
          </PromptInputProvider>
        </SidebarInset>
      </SidebarProvider>
      <CommandPalette />
    </>
  );
}
