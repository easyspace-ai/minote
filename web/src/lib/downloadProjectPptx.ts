import { apiOrigin, mergeAuthHeaders } from "@/api/http";

/** GET authenticated PPTX blob and trigger browser download. */
export async function downloadProjectPptx(
  projectId: string,
  materialId: number,
  filename: string,
): Promise<void> {
  const url = `${apiOrigin()}/api/v1/projects/${projectId}/materials/${materialId}/pptx`;
  const res = await fetch(url, { headers: mergeAuthHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  const blob = await res.blob();
  const a = document.createElement("a");
  const href = URL.createObjectURL(blob);
  a.href = href;
  a.download = filename || "slides.pptx";
  a.click();
  URL.revokeObjectURL(href);
}
