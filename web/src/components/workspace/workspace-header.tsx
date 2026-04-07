"use client";

import { MessageSquarePlus } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { useI18n } from "@/core/i18n/hooks";
import { env } from "@/env";
import { cn } from "@/lib/utils";

export function WorkspaceHeader({ className }: { className?: string }) {
  const { t } = useI18n();
  const { state } = useSidebar();
  const { pathname } = useLocation();
  return (
    <>
      <div
        className={cn(
          "group/workspace-header flex h-12 flex-col justify-center",
          className,
        )}
      >
        {state === "collapsed" ? (
          <div className="group-has-data-[collapsible=icon]/sidebar-wrapper:-translate-y flex w-full cursor-pointer items-center justify-center">
            <Link to="/" className="block pt-1 group-hover/workspace-header:hidden" title="MetaNote">
              <img
                src="/logo.jpg"
                alt=""
                className="h-9 w-9 rounded-full object-cover ring-1 ring-black/[0.06]"
              />
            </Link>
            <SidebarTrigger className="hidden pl-2 group-hover/workspace-header:block" />
          </div>
        ) : (
          <div className="flex items-center justify-between gap-2">
            {env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true" ? (
              <Link to="/" className="ml-2 flex min-w-0 items-center gap-2" title="MetaNote">
                <img
                  src="/logo.jpg"
                  alt=""
                  className="h-9 w-9 shrink-0 rounded-full object-cover ring-1 ring-black/[0.06]"
                />
                <span className="text-foreground truncate font-semibold tracking-tight">MetaNote</span>
              </Link>
            ) : (
              <div className="ml-2 flex min-w-0 cursor-default items-center gap-2">
                <img
                  src="/logo.jpg"
                  alt=""
                  className="h-9 w-9 shrink-0 rounded-full object-cover ring-1 ring-black/[0.06]"
                />
                <span className="text-foreground truncate font-semibold tracking-tight">MetaNote</span>
              </div>
            )}
            <SidebarTrigger />
          </div>
        )}
      </div>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            isActive={pathname === "/workspace/chats/new"}
            asChild
          >
            <Link className="text-muted-foreground" to="/workspace/chats/new">
              <MessageSquarePlus size={16} />
              <span>{t.sidebar.newChat}</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </>
  );
}
