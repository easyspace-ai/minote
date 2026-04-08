import { useCallback, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import { mergeAuthHeaders } from "@/api/http";

type SSEHandlers = {
  onChunk?: (content: string) => void;
  onStart?: (requestId: string) => void;
  onDone?: () => void;
};

export type SendStreamOptions = {
  /** Stable id for parallel studio jobs; must match backend chunk routing when set. */
  requestId?: string;
  /** When set (web-ui-studio), backend limits inlined library full-text to these document ids. */
  studioDocumentIds?: number[];
};

export function useChatSend() {
  const [inFlight, setInFlight] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const sendStream = useCallback(
    async (
      conversationId: number,
      content: string,
      tabId: string,
      handlers: SSEHandlers,
      options?: SendStreamOptions,
    ) => {
      const requestId = options?.requestId ?? uuidv4();
      setInFlight((n) => n + 1);
      setError(null);
      const base = import.meta.env.DEV ? "" : import.meta.env.VITE_BACKEND_ORIGIN ?? "";
      /** Backend sends accumulated assistant text per chunk event; expose deltas to callers. */
      let accumulated = "";
      try {
        const res = await fetch(`${base}/api/v1/chat/messages?stream=1`, {
          method: "POST",
          headers: mergeAuthHeaders({
            "Content-Type": "application/json",
            Accept: "text/event-stream",
          }),
          body: JSON.stringify({
            conversation_id: conversationId,
            content,
            tab_id: tabId,
            request_id: requestId,
            ...(options?.studioDocumentIds != null && options.studioDocumentIds.length > 0
              ? { studio_document_ids: options.studioDocumentIds }
              : {}),
          }),
        });
        if (!res.ok || !res.body) {
          const text = await res.text();
          // 识别会话不存在的错误
          if (res.status === 404 || text.includes("conversation") || text.includes("会话")) {
            throw new Error(`会话已失效或不存在，请刷新页面重试 (${res.status})`);
          }
          throw new Error(text || `请求失败 (${res.status})`);
        }
        const reader = res.body.getReader();
        const dec = new TextDecoder();
        let buf = "";
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
              const delta = full.slice(accumulated.length);
              accumulated = full;
              if (delta) handlers.onChunk?.(delta);
            }
            if (ev === "start" && typeof payload.request_id === "string") {
              handlers.onStart?.(payload.request_id);
            }
            if (ev === "done") {
              handlers.onDone?.();
            }
            if (ev === "error") {
              throw new Error(String(payload.message ?? "stream_error"));
            }
          }
        }
        handlers.onDone?.();
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        setError(msg);
        throw e;
      } finally {
        setInFlight((n) => Math.max(0, n - 1));
      }
    },
    [],
  );

  return { sendStream, streaming: inFlight > 0, inFlight, error, setError };
}
