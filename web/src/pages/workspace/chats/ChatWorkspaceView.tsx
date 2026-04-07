"use client";

import { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  PromptInputProvider,
  type PromptInputMessage,
} from "@/components/ai-elements/prompt-input";
import type { PromptInputFilePart } from "@/core/uploads/prompt-input-files";
import { ArtifactsProvider } from "@/components/workspace/artifacts";
import { ChatBox, useSpecificChatMode } from "@/components/workspace/chats";
import { ExportTrigger } from "@/components/workspace/export-trigger";
import { InputBox } from "@/components/workspace/input-box";
import {
  MessageList,
  MESSAGE_LIST_DEFAULT_PADDING_BOTTOM,
  MESSAGE_LIST_FOLLOWUPS_EXTRA_PADDING_BOTTOM,
} from "@/components/workspace/messages";
import { ThreadContext } from "@/components/workspace/messages/context";
import { ThreadTitle } from "@/components/workspace/thread-title";
import { TodoList } from "@/components/workspace/todo-list";
import { TokenUsageIndicator } from "@/components/workspace/token-usage-indicator";
import { Welcome } from "@/components/workspace/welcome";
import { useI18n } from "@/core/i18n/hooks";
import { useNotification } from "@/core/notification/hooks";
import { useThreadSettings } from "@/core/settings";
import { useThreadStream } from "@/core/threads/hooks";
import { textOfMessage } from "@/core/threads/utils";
import { env } from "@/env";
import { claimInitialThreadSeed, releaseInitialThreadSeed } from "@/lib/initialThreadSeed";
import { cn } from "@/lib/utils";

/** Count valid document ids in LangGraph run context (project notebook checked materials). */
function studioDocumentIdsCount(extra: Record<string, unknown> | undefined): number {
  if (!extra) return 0;
  const raw = extra.studio_document_ids;
  if (!Array.isArray(raw) || raw.length === 0) return 0;
  let n = 0;
  for (const x of raw) {
    if (typeof x === "number" && Number.isFinite(x) && x > 0) n++;
    else if (typeof x === "string") {
      const t = x.trim();
      if (t !== "" && /^\d+$/.test(t)) n++;
    }
  }
  return n;
}

export type ChatWorkspaceViewProps = {
  threadId: string;
  isNewThread: boolean;
  setIsNewThread: (v: boolean) => void;
  isMock?: boolean;
  /** When false (default), replace URL on first message (standalone /workspace/chats/...). */
  replaceUrlOnStart?: boolean;
  replaceUrlPath?: string;
  /** After thread is ready, send this user message once (e.g. from /new landing page). */
  seedUserMessage?: string | null;
  /** Same navigation flow: attachments to upload with the seed message. */
  seedUserFiles?: File[];
  /** Apply once before the seed send (e.g. Pro from new-project page). */
  seedChatMode?: "flash" | "thinking" | "pro" | "ultra";
  /** Merged into LangGraph run `context` (e.g. project notebook `studio_document_ids`). */
  streamExtraContext?: Record<string, unknown>;
  /**
   * When true, do not wrap with ArtifactsProvider (parent already provides it, e.g. project notebook).
   */
  skipArtifactsProvider?: boolean;
};

export function ChatWorkspaceView(props: ChatWorkspaceViewProps) {
  const { skipArtifactsProvider, ...rest } = props;
  const inner = (
    <PromptInputProvider>
      <ChatWorkspaceViewInner {...rest} />
    </PromptInputProvider>
  );
  if (skipArtifactsProvider) {
    return inner;
  }
  return <ArtifactsProvider>{inner}</ArtifactsProvider>;
}

function ChatWorkspaceViewInner({
  threadId,
  isNewThread,
  setIsNewThread,
  isMock = false,
  replaceUrlOnStart = true,
  replaceUrlPath,
  seedUserMessage = null,
  seedUserFiles,
  seedChatMode,
  streamExtraContext,
}: ChatWorkspaceViewProps) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [showFollowups, setShowFollowups] = useState(false);
  const [settings, setSettings] = useThreadSettings(threadId);
  const [mounted, setMounted] = useState(false);
  const [seedContextReady, setSeedContextReady] = useState(() => !seedChatMode);
  useSpecificChatMode();

  useEffect(() => {
    setMounted(true);
  }, []);

  useLayoutEffect(() => {
    if (!threadId) {
      setSeedContextReady(true);
      return;
    }
    if (seedChatMode) {
      setSettings("context", { mode: seedChatMode });
    }
    setSeedContextReady(true);
  }, [threadId, seedChatMode, setSettings]);

  const { showNotification } = useNotification();

  const streamContextArg = useMemo(() => {
    if (!streamExtraContext || Object.keys(streamExtraContext).length === 0) {
      return undefined;
    }
    return streamExtraContext;
  }, [streamExtraContext]);

  const attachedStudioDocCount = useMemo(
    () => studioDocumentIdsCount(streamContextArg),
    [streamContextArg],
  );

  const [thread, sendMessage, isUploading] = useThreadStream({
    threadId: isNewThread ? undefined : threadId,
    context: settings.context,
    isMock,
    onStart: () => {
      setIsNewThread(false);
      if (replaceUrlOnStart) {
        const path = replaceUrlPath ?? `/workspace/chats/${threadId}`;
        void navigate(path, { replace: true });
      }
    },
    onFinish: (state) => {
      if (document.hidden || !document.hasFocus()) {
        let body = "Conversation finished";
        const lastMessage = state.messages.at(-1);
        if (lastMessage) {
          const textContent = textOfMessage(lastMessage);
          if (textContent) {
            body =
              textContent.length > 200
                ? textContent.substring(0, 200) + "..."
                : textContent;
          }
        }
        showNotification(state.title, { body });
      }
    },
  });

  const handleSubmit = useCallback(
    (message: PromptInputMessage) => {
      void sendMessage(threadId, message, streamContextArg);
    },
    [sendMessage, streamContextArg, threadId],
  );
  const handleStop = useCallback(async () => {
    await thread.stop();
  }, [thread]);

  useEffect(() => {
    if (!seedContextReady || !threadId) return;
    const raw = seedUserMessage?.trim() ?? "";
    const files = seedUserFiles ?? [];
    if (!raw && files.length === 0) return;
    const fileKey = files.map((f) => `${f.name}:${f.size}`).join("|");
    if (!claimInitialThreadSeed(threadId, raw, fileKey)) return;

    const parts: PromptInputFilePart[] = files.map((file) => ({
      type: "file",
      url: URL.createObjectURL(file),
      mediaType: file.type || "application/octet-stream",
      filename: file.name,
      file,
    }));

    void (async () => {
      try {
        await sendMessage(
          threadId,
          { text: raw, files: parts },
          streamContextArg,
        );
      } catch {
        releaseInitialThreadSeed(threadId, raw, fileKey);
      } finally {
        for (const p of parts) {
          if (p.url.startsWith("blob:")) {
            URL.revokeObjectURL(p.url);
          }
        }
      }
    })();
  }, [
    seedContextReady,
    threadId,
    seedUserMessage,
    seedUserFiles,
    sendMessage,
    streamContextArg,
  ]);

  const messageListPaddingBottom = showFollowups
    ? MESSAGE_LIST_DEFAULT_PADDING_BOTTOM +
      MESSAGE_LIST_FOLLOWUPS_EXTRA_PADDING_BOTTOM
    : undefined;

  return (
    <ThreadContext.Provider value={{ thread, isMock }}>
      <ChatBox threadId={threadId}>
        <div className="relative flex size-full min-h-0 justify-between">
          <header
            className={cn(
              "absolute top-0 right-0 left-0 z-30 flex h-12 shrink-0 items-center px-4",
              isNewThread
                ? "bg-background/0 backdrop-blur-none"
                : "bg-background/80 shadow-xs backdrop-blur",
            )}
          >
            <div className="flex w-full items-center text-sm font-medium">
              <ThreadTitle threadId={threadId} thread={thread} />
            </div>
            <div className="flex items-center gap-2">
              <TokenUsageIndicator messages={thread.messages} />
              <ExportTrigger threadId={threadId} />
            </div>
          </header>
          <main className="flex min-h-0 max-w-full grow flex-col">
            <div className="flex size-full justify-center">
              <MessageList
                className={cn("size-full", !isNewThread && "pt-10")}
                threadId={threadId}
                thread={thread}
                paddingBottom={messageListPaddingBottom}
              />
            </div>
            <div className="absolute right-0 bottom-0 left-0 z-30 flex justify-center px-4">
              <div
                className={cn(
                  "relative w-full",
                  isNewThread && "-translate-y-[calc(50vh-96px)]",
                  isNewThread
                    ? "max-w-(--container-width-sm)"
                    : "max-w-(--container-width-md)",
                )}
              >
                {/* {attachedStudioDocCount > 0 ? (
                  <p className="text-muted-foreground mb-1.5 px-2 text-center text-[11px] leading-snug">
                    已勾选 {attachedStudioDocCount}{" "}
                    份资料：发送消息时将自动附上全文作为上下文，无需 @ 文件名。
                  </p>
                ) : null} */}
                <div className="absolute -top-4 right-0 left-0 z-0">
                  <div className="absolute right-0 bottom-0 left-0">
                    <TodoList
                      className="bg-background/5"
                      todos={thread.values.todos ?? []}
                      hidden={
                        !thread.values.todos || thread.values.todos.length === 0
                      }
                    />
                  </div>
                </div>
                {mounted ? (
                  <InputBox
                    className={cn("bg-background/5 w-full -translate-y-4")}
                    isNewThread={isNewThread}
                    threadId={threadId}
                    autoFocus={isNewThread}
                    status={
                      thread.error
                        ? "error"
                        : thread.isLoading
                          ? "streaming"
                          : "ready"
                    }
                    context={settings.context}
                    extraHeader={
                      isNewThread && <Welcome mode={settings.context.mode} />
                    }
                    disabled={
                      env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true" ||
                      isUploading
                    }
                    onContextChange={(context) =>
                      setSettings("context", context)
                    }
                    onFollowupsVisibilityChange={setShowFollowups}
                    onSubmit={handleSubmit}
                    onStop={handleStop}
                  />
                ) : (
                  <div
                    aria-hidden="true"
                    className={cn(
                      "bg-background/5 h-32 w-full -translate-y-4 rounded-2xl border",
                    )}
                  />
                )}
                {env.NEXT_PUBLIC_STATIC_WEBSITE_ONLY === "true" && (
                  <div className="text-muted-foreground/67 w-full translate-y-12 text-center text-xs">
                    {t.common.notAvailableInDemoMode}
                  </div>
                )}
              </div>
            </div>
          </main>
        </div>
      </ChatBox>
    </ThreadContext.Provider>
  );
}
