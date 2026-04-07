import type { ChatTransport, UIMessage, UIMessageChunk } from "ai";
import { apiOrigin, mergeAuthHeaders } from "@/api/http";

/** Resolved library file for POST /api/v1/chat/messages `images` field (matches backend ImagePayload). */
export type LibraryChatAttachmentPayload = {
  document_id: number;
  original_name: string;
  mime_type: string;
  base64: string;
  size: number;
};

export type ProjectChatMetadata = {
  libraryAttachments?: LibraryChatAttachmentPayload[];
};

export type ProjectUIMessage = UIMessage<ProjectChatMetadata>;

function getUserText(m: ProjectUIMessage): string {
  return m.parts
    .filter((p): p is { type: "text"; text: string } => p.type === "text")
    .map((p) => p.text)
    .join("");
}

function buildImagesPayload(meta: ProjectChatMetadata | undefined): Record<string, unknown>[] {
  const list = meta?.libraryAttachments ?? [];
  return list.map((a) => ({
    kind: "file",
    source: "inline_base64",
    mime_type: a.mime_type,
    base64: a.base64,
    file_name: a.original_name,
    size: a.size,
  }));
}

export function createChatclawChatTransport(options: {
  getConversationId: () => number | null;
  tabId: string;
}): ChatTransport<ProjectUIMessage> {
  const { getConversationId, tabId } = options;

  return {
    async sendMessages({ messages, abortSignal, trigger }) {
      if (trigger === "regenerate-message") {
        throw new Error("Regeneration is not supported for this workspace chat.");
      }

      const convId = getConversationId();
      if (convId == null || convId <= 0) {
        throw new Error("Conversation is not ready.");
      }

      const lastUser = [...messages].reverse().find((m) => m.role === "user");
      if (!lastUser) {
        throw new Error("No user message to send.");
      }

      const content = getUserText(lastUser).trim();
      const images = buildImagesPayload(lastUser.metadata);
      if (!content && images.length === 0) {
        throw new Error("Message content or attachment is required.");
      }

      const base = apiOrigin();
      const res = await fetch(`${base}/api/v1/chat/messages?stream=1`, {
        method: "POST",
        headers: mergeAuthHeaders({
          "Content-Type": "application/json",
          Accept: "text/event-stream",
        }),
        body: JSON.stringify({
          conversation_id: convId,
          content,
          tab_id: tabId,
          ...(images.length > 0 ? { images } : {}),
        }),
        signal: abortSignal,
      });

      if (!res.ok || !res.body) {
        const t = await res.text();
        throw new Error(t || "Chat request failed.");
      }

      const textId = "text-1";

      return new ReadableStream<UIMessageChunk>({
        async start(controller) {
          controller.enqueue({ type: "start" });
          controller.enqueue({ type: "start-step" });
          controller.enqueue({ type: "text-start", id: textId });

          const reader = res.body!.getReader();
          const dec = new TextDecoder();
          let buf = "";
          let acc = "";
          let streamError: string | null = null;

          try {
            for (;;) {
              const { value, done } = await reader.read();
              if (done) break;
              buf += dec.decode(value, { stream: true });
              const blocks = buf.split("\n\n");
              buf = blocks.pop() ?? "";
              for (const block of blocks) {
                let ev = "";
                let data = "";
                for (const line of block.split("\n")) {
                  if (line.startsWith("event:")) ev = line.slice(6).trim();
                  if (line.startsWith("data:")) data = line.slice(5).trim();
                }
                if (!data) continue;
                const payload = JSON.parse(data) as Record<string, unknown>;
                if (ev === "chunk" && typeof payload.content === "string") {
                  const full = payload.content;
                  const delta = full.slice(acc.length);
                  acc = full;
                  if (delta) {
                    controller.enqueue({ type: "text-delta", id: textId, delta });
                  }
                }
                if (ev === "error") {
                  streamError = String(payload.message ?? "stream_error");
                  break;
                }
              }
              if (streamError) break;
            }

            if (streamError) {
              controller.enqueue({ type: "error", errorText: streamError });
            } else {
              controller.enqueue({ type: "text-end", id: textId });
              controller.enqueue({ type: "finish-step" });
              controller.enqueue({ type: "finish" });
            }
          } catch (e) {
            controller.enqueue({
              type: "error",
              errorText: e instanceof Error ? e.message : String(e),
            });
          } finally {
            reader.releaseLock();
            controller.close();
          }
        },
      });
    },

    async reconnectToStream() {
      return null;
    },
  };
}
