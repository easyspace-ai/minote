"use client";

import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import { env } from "@/env";

/**
 * Standalone workspace index: send users to a new chat (or demo thread when static-only).
 */
export function WorkspaceHomePage() {
  const navigate = useNavigate();

  useEffect(() => {
    if (env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true") {
      void fetch("/demo/threads/")
        .then((r) => r.text())
        .catch(() => null);
    }
    navigate("/workspace/chats/new", { replace: true });
  }, [navigate]);

  return (
    <div className="text-muted-foreground flex h-full items-center justify-center text-sm">
      Redirecting…
    </div>
  );
}
