/** Dev: Vite proxy. Prod: set `VITE_BACKEND_ORIGIN`. */
export function apiOrigin(): string {
  if (import.meta.env.DEV) return "";
  return import.meta.env.VITE_BACKEND_ORIGIN ?? "";
}

let authTokenGetter: () => string | null = () => null;
let onUnauthorized: (() => void) | null = null;

/** Called from AuthProvider so apiFetch can attach Bearer tokens. */
export function setAuthTokenGetter(fn: () => string | null) {
  authTokenGetter = fn;
}

/** Optional: clear session and redirect when API returns 401 with a token sent. */
export function setOnUnauthorized(fn: (() => void) | null) {
  onUnauthorized = fn;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** Merge optional headers with Bearer auth (for fetch/SSE outside apiFetch). */
export function mergeAuthHeaders(init?: HeadersInit): Headers {
  const headers = new Headers(init);
  const tok = authTokenGetter();
  if (tok) headers.set("Authorization", `Bearer ${tok}`);
  return headers;
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${apiOrigin()}${path}`;
  const headers = mergeAuthHeaders(init?.headers);
  if (!headers.has("Accept")) headers.set("Accept", "application/json");

  const res = await fetch(url, {
    ...init,
    headers,
  });
  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text) as unknown;
    } catch {
      data = text;
    }
  }
  if (!res.ok) {
    if (res.status === 401 && authTokenGetter()) {
      onUnauthorized?.();
    }
    const msg =
      typeof data === "object" && data !== null && "error" in data
        ? String((data as { error: string }).error)
        : text || res.statusText;
    throw new ApiError(msg, res.status, data);
  }
  return data as T;
}
