import { useChat } from "@ai-sdk/react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef } from "react";
import { chatclawApi } from "@/api/chatclaw";
import { StreamingIndicator } from "@/components/layout/NotebookShell";
import { MessageList } from "@/components/messages/message-list";
import {
  createChatclawChatTransport,
  type ProjectUIMessage,
} from "@/lib/chatclawChatTransport";
import { ChatComposerWithLibrary, type LibraryDocRow } from "./ChatComposerWithLibrary";

export function ProjectChatDock(props: {
  convId: number | null;
  agentId: number | null;
  docs: LibraryDocRow[];
}) {
  const { convId, agentId, docs } = props;
  const qc = useQueryClient();
  const scrollRef = useRef<HTMLDivElement>(null);

  const messagesQuery = useQuery({
    queryKey: ["messages", convId],
    queryFn: () => chatclawApi.conversations.messages(convId!),
    enabled: convId != null && convId > 0,
  });

  const convRef = useRef(convId);
  convRef.current = convId;

  const transport = useMemo(
    () =>
      createChatclawChatTransport({
        getConversationId: () => convRef.current,
        tabId: "web-ui",
      }),
    [],
  );

  const { messages, sendMessage, status, setMessages, error } = useChat<ProjectUIMessage>({
    id: convId != null ? `notebook-conv-${convId}` : "notebook-pending",
    transport,
    messages: [],
    onFinish: () => {
      const c = convRef.current;
      if (c != null) {
        void qc.invalidateQueries({ queryKey: ["messages", c] });
      }
      if (agentId != null) {
        void qc.invalidateQueries({ queryKey: ["conversations", agentId] });
      }
    },
  });

  useEffect(() => {
    if (status !== "ready" || convId == null) return;
    const rows = messagesQuery.data;
    if (!rows) return;
    setMessages(
      rows.map((m) => ({
        id: String(m.id),
        role: m.role as "user" | "assistant",
        parts: [{ type: "text" as const, text: m.content }],
      })),
    );
  }, [convId, messagesQuery.data, status, setMessages]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [messages, status]);

  async function handleSubmit({
    text,
    documentIds,
  }: {
    text: string;
    documentIds: number[];
  }) {
    if (!convId) return;
    const trimmed = text.trim();
    const finalText =
      trimmed || (documentIds.length > 0 ? "请分析附件内容。" : "");
    if (!finalText && documentIds.length === 0) return;

    const libraryAttachments =
      documentIds.length > 0
        ? await Promise.all(
            documentIds.map(async (id) => {
              const a = await chatclawApi.documents.chatAttachment(id);
              return {
                document_id: id,
                original_name: a.original_name,
                mime_type: a.mime_type,
                base64: a.base64_data,
                size: a.file_size,
              };
            }),
          )
        : undefined;

    await sendMessage({
      text: finalText,
      metadata: libraryAttachments?.length ? { libraryAttachments } : undefined,
    });
  }

  const busy = status === "submitted" || status === "streaming";

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      {!convId && <p className="text-sm text-muted-foreground p-6">正在准备会话…</p>}
      {convId && (
        <MessageList
          messages={messages}
          isLoading={busy}
          className="flex-1 pt-0"
        />
      )}
      {error && (
        <p className="shrink-0 border-t border-outline-variant/10 bg-popover px-6 py-2 text-xs text-on-surface shadow-sm dark:shadow-none dark:ring-1 dark:ring-white/10">
          发送失败：{error.message}
        </p>
      )}
      <ChatComposerWithLibrary
        docs={docs}
        disabled={!convId || busy}
        busy={busy}
        placeholder={convId ? "针对资料提问，输入 @ 引用上传的文档…" : "等待会话…"}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
