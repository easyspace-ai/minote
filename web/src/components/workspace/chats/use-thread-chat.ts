"use client";

import { useParams, useLocation, useSearchParams } from "react-router-dom";
import { useEffect, useState } from "react";

import { uuid } from "@/core/utils/uuid";

export function useThreadChat() {
  const { thread_id: threadIdFromPath } = useParams<{ thread_id: string }>();
  const { pathname } = useLocation();

  const [searchParams] = useSearchParams();
  const [threadId, setThreadId] = useState(() => {
    if (threadIdFromPath === "new" || !threadIdFromPath) {
      return uuid();
    }
    return threadIdFromPath;
  });

  const [isNewThread, setIsNewThread] = useState(
    () => threadIdFromPath === "new" || !threadIdFromPath,
  );

  useEffect(() => {
    if (pathname.endsWith("/new")) {
      setIsNewThread(true);
      setThreadId(uuid());
    }
  }, [pathname]);
  const isMock = searchParams.get("mock") === "true";
  return { threadId, isNewThread, setIsNewThread, isMock };
}
