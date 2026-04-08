import { chatclawApi } from "@/api/chatclaw";

export type WaitForStudioSlidesArtifactOptions = {
  /** Total time to poll before giving up (agent may still be writing .pptx after SSE ends). */
  timeoutMs?: number;
  intervalMs?: number;
};

const defaultTimeoutMs = 180_000;
const defaultIntervalMs = 2000;

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

/**
 * Polls Notex until the conversation's LangGraph thread lists a skill .pptx artifact (strict Agent+skill path).
 * Call after the studio chat stream completes and before POST .../materials/slides-pptx.
 */
export async function waitForStudioSlidesArtifact(
  conversationId: number,
  options?: WaitForStudioSlidesArtifactOptions,
): Promise<void> {
  const timeoutMs = options?.timeoutMs ?? defaultTimeoutMs;
  const intervalMs = options?.intervalMs ?? defaultIntervalMs;
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    const { ready } = await chatclawApi.conversations.studioSlidesArtifactStatus(conversationId);
    if (ready) {
      return;
    }
    if (Date.now() >= deadline) {
      throw new Error(
        "演示文稿仍未就绪：Agent 须在本会话环境中写出真实 .pptx（如 user-data/outputs/）并出现在线程文件列表。" +
          " 若模型只返回了文字或 HTML，请重试并确认已启用演示文稿/PPT 相关 skill。",
      );
    }
    await sleep(intervalMs);
  }
}
