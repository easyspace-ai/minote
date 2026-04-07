"use client";

import { useThreadChat } from "@/components/workspace/chats";

import { ChatWorkspaceView } from "./ChatWorkspaceView";

export function ChatThreadPage() {
  const { threadId, isNewThread, setIsNewThread, isMock } = useThreadChat();
  return (
    <ChatWorkspaceView
      threadId={threadId}
      isNewThread={isNewThread}
      setIsNewThread={setIsNewThread}
      isMock={isMock}
    />
  );
}
