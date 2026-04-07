import { apiOrigin, mergeAuthHeaders } from "@/api/http";

/** Volcengine openspeech 请求体过长易失败；控制在约 4k 字符内。 */
export const STUDIO_TTS_MAX_CHARS = 3800;

export function truncateForTTS(text: string, maxChars: number = STUDIO_TTS_MAX_CHARS): { text: string; truncated: boolean } {
  const t = text.trim();
  if (t.length <= maxChars) {
    return { text: t, truncated: false };
  }
  return { text: t.slice(0, maxChars) + "\n\n…[口播稿已截断以适配语音合成长度上限]", truncated: true };
}

/**
 * Calls POST /api/tts (unified notex + LangGraph server; Vite proxies /api in dev).
 * Requires VOLCENGINE_TTS_API_KEY or TTS_API_KEY on the server (see .env.example).
 * Uses an abort timeout so a stuck TTS does not leave Studio in “正在生成…” forever.
 */
const STUDIO_TTS_FETCH_MS = 180_000;

export async function synthesizePodcastSpeechMp3(plainText: string): Promise<ArrayBuffer> {
  const { text } = truncateForTTS(plainText);
  if (!text.trim()) {
    throw new Error("口播稿为空，无法合成语音");
  }
  const url = `${apiOrigin()}/api/tts`;
  const ac = new AbortController();
  const timer = setTimeout(() => ac.abort(), STUDIO_TTS_FETCH_MS);
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      signal: ac.signal,
      headers: mergeAuthHeaders({
        "Content-Type": "application/json",
        Accept: "audio/mpeg",
      }),
      body: JSON.stringify({
        text,
        input: text,
        format: "mp3",
        response_format: "mp3",
      }),
    });
  } catch (e) {
    if (e instanceof Error && e.name === "AbortError") {
      throw new Error(
        `语音合成超时（>${Math.round(STUDIO_TTS_FETCH_MS / 1000)}s）。请缩短口播稿或检查服务端火山 TTS 网络与 VOLCENGINE_TTS_API_KEY / TTS_API_KEY。`,
      );
    }
    throw e;
  } finally {
    clearTimeout(timer);
  }
  if (!res.ok) {
    let detail = await res.text();
    try {
      const j = JSON.parse(detail) as { detail?: string };
      if (typeof j.detail === "string") detail = j.detail;
    } catch {
      /* keep text */
    }
    throw new Error(detail || `TTS HTTP ${res.status}`);
  }
  return res.arrayBuffer();
}

export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]!);
  }
  return btoa(binary);
}
