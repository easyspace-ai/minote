import { apiOrigin, mergeAuthHeaders } from "@/api/http";

/** GET authenticated HTML export (same URL as download; use for iframe srcDoc preview). */
export async function fetchStudioFileText(projectId: number, materialId: number): Promise<string> {
  const url = `${apiOrigin()}/api/v1/projects/${projectId}/materials/${materialId}/studio-file`;
  const res = await fetch(url, { headers: mergeAuthHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  return res.text();
}

/** GET authenticated studio export as Blob (HTML, audio, etc.). */
export async function fetchStudioFileBlob(projectId: number, materialId: number): Promise<Blob> {
  const url = `${apiOrigin()}/api/v1/projects/${projectId}/materials/${materialId}/studio-file`;
  const res = await fetch(url, { headers: mergeAuthHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  return res.blob();
}

/** GET authenticated studio file blob and trigger browser download. */
export async function downloadProjectStudioFile(
  projectId: number,
  materialId: number,
  filename: string,
): Promise<void> {
  const url = `${apiOrigin()}/api/v1/projects/${projectId}/materials/${materialId}/studio-file`;
  const res = await fetch(url, { headers: mergeAuthHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  const blob = await res.blob();
  const a = document.createElement("a");
  const href = URL.createObjectURL(blob);
  a.href = href;
  a.download = filename || "export.html";
  a.click();
  URL.revokeObjectURL(href);
}
