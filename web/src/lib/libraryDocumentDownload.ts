import { chatclawApi } from "@/api/chatclaw";

function base64ToBlob(base64: string, mime: string): Blob {
  const binary = atob(base64);
  const len = binary.length;
  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i++) bytes[i] = binary.charCodeAt(i);
  return new Blob([bytes], { type: mime || "application/octet-stream" });
}

function effectiveMime(originalName: string, mime: string): string {
  const m = (mime || "").split(";")[0]!.trim().toLowerCase();
  if (m && m !== "application/octet-stream") return mime.split(";")[0]!.trim();
  const lower = originalName.toLowerCase();
  if (lower.endsWith(".pdf")) return "application/pdf";
  if (lower.endsWith(".md") || lower.endsWith(".markdown")) return "text/markdown";
  if (lower.endsWith(".html") || lower.endsWith(".htm")) return "text/html";
  if (lower.endsWith(".txt")) return "text/plain";
  if (lower.endsWith(".json")) return "application/json";
  if (lower.endsWith(".png")) return "image/png";
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  return "application/octet-stream";
}

/** 触发浏览器下载资料库原始文件（与 chat-attachment 同源）。 */
export async function downloadLibraryDocument(documentId: number, fallbackName: string): Promise<void> {
  const att = await chatclawApi.documents.chatAttachment(documentId);
  if (!att.base64_data) throw new Error("无文件数据");
  const name = att.original_name?.trim() || fallbackName;
  const mime = effectiveMime(name, att.mime_type);
  const blob = base64ToBlob(att.base64_data, mime);
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}
