import { apiFetch } from "./http";

export type Agent = {
  id: number;
  name: string;
  prompt: string;
  icon: string;
  openclaw_agent_id: string;
  default_llm_provider_id: string;
  default_llm_model_id: string;
  llm_temperature: number;
  llm_top_p: number;
  llm_max_context_count: number;
  llm_max_tokens: number;
  enable_llm_temperature: boolean;
  enable_llm_top_p: boolean;
  enable_llm_max_tokens: boolean;
  retrieval_match_threshold: number;
  retrieval_top_k: number;
  sandbox_mode: string;
  sandbox_network: boolean;
  work_dir: string;
  mcp_enabled: boolean;
  /** JSON array string, e.g. '["uuid"]' */
  mcp_server_ids: string;
  mcp_server_enabled_ids: string;
  created_at?: string;
  updated_at?: string;
};

export type Library = {
  id: number;
  name: string;
  chunk_size: number;
  chunk_overlap: number;
};

/** Persisted Studio「生成范围与自定义」(per project, server jsonb). */
export type StudioScopeSettings = {
  includeChat: boolean;
  chatSummaryOnly: boolean;
  chatMaxMessages: string;
  customExtra: string;
};

export type Project = {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
  description: string;
  category: string;
  library_id: number;
  starred?: boolean;
  archived?: boolean;
  icon_index?: number;
  accent_hex?: string;
  studio_scope?: StudioScopeSettings;
};

export type ProjectPatchBody = {
  name?: string;
  description?: string;
  category?: string;
  starred?: boolean;
  archived?: boolean;
  icon_index?: number;
  accent_hex?: string;
  studio_scope?: StudioScopeSettings;
};

export type Conversation = {
  id: number;
  agent_id: number;
  name: string;
  last_message: string;
  library_ids: number[];
  chat_mode: string;
  /** LangGraph thread id when bound (notex). */
  thread_id?: string;
  /** Hidden session used only for Studio generations (not listed in sidebar). */
  studio_only?: boolean;
};

export type ConversationCreateBody = {
  agent_id: number;
  name?: string;
  library_ids?: number[];
  chat_mode?: string;
  studio_only?: boolean;
};

export type ConversationPatchBody = {
  name: string;
};

export type Message = {
  id: number;
  conversation_id: number;
  role: string;
  content: string;
  status: string;
};

export type ProviderDTO = {
  id: number;
  provider_id: string;
  name: string;
  type: string;
  icon: string;
  is_builtin: boolean;
  is_free: boolean;
  enabled: boolean;
  sort_order: number;
  api_endpoint: string;
  api_key: string;
  extra_config: string;
  created_at: string;
  updated_at: string;
};

export type ProviderPatchBody = {
  enabled?: boolean;
  api_key?: string;
  api_endpoint?: string;
  extra_config?: string;
};

export type ModelDTO = {
  id: number;
  provider_id: string;
  model_id: string;
  name: string;
  type: string;
  capabilities: string[];
  is_builtin: boolean;
  enabled: boolean;
  sort_order: number;
};

export type ModelGroupDTO = {
  type: string;
  models: ModelDTO[];
};

export type ProviderWithModelsDTO = {
  provider: ProviderDTO;
  model_groups: ModelGroupDTO[];
};

export type InstalledSkill = {
  slug: string;
  name: string;
  description: string;
  version: string;
  source: string;
  enabled: boolean;
  installedAt: string;
};

/** Studio artifact kinds (must match backend project materials). */
export type StudioMaterialKind =
  | "audio"
  | "slides"
  | "html"
  | "mindmap"
  | "report"
  | "infographic"
  | "quiz"
  | "data_table";

/** `processing` = server-side in-flight (e.g. legacy studio/create); list UI should treat like pending. */
export type StudioMaterialStatus = "pending" | "processing" | "ready" | "failed";

export type StudioMaterial = {
  id: number;
  created_at: string;
  updated_at: string;
  project_id: string;
  kind: StudioMaterialKind;
  title: string;
  status: StudioMaterialStatus;
  subtitle: string;
  /** Arbitrary JSON from backend (audioUrl, markdown, slideUrls, etc.) */
  payload: Record<string, unknown>;
};

export type CreateStudioMaterialBody = {
  kind: StudioMaterialKind;
  title: string;
  status?: StudioMaterialStatus;
  subtitle?: string;
  payload?: Record<string, unknown>;
};

export type PatchStudioMaterialBody = {
  status?: StudioMaterialStatus;
  title?: string;
  subtitle?: string;
  payload?: Record<string, unknown>;
};

export type VectorInspectNode = {
  node_id: number;
  level: number;
  chunk_order: number;
  has_vector: boolean;
  content_preview: string;
  vector_preview?: number[];
};

export type DocumentChatAttachment = {
  base64_data: string;
  mime_type: string;
  original_name: string;
  file_size: number;
};

/** Document row returned by PATCH /api/v1/documents/:id (no file body). */
export type LibraryDocumentRecord = {
  id: number;
  library_id: number;
  original_name: string;
  file_size: number;
  mime_type?: string;
  created_at?: string;
  starred?: boolean;
};

export type VectorInspectResult = {
  document_id: number;
  library_id: number;
  original_name: string;
  parsing_status: number;
  embedding_status: number;
  word_total: number;
  split_total: number;
  embedding_dimension: number;
  node_rows: number;
  vectors_indexed: number;
  nodes: VectorInspectNode[];
};

export type MetaResponse = {
  run_mode: string;
  auth_required?: boolean;
  has_users?: boolean;
};

export type AuthUser = {
  id: number;
  email: string;
  created_at: string;
  /** Optional; from extended APIs */
  username?: string;
  credits_balance?: number;
  credits_used?: number;
};

export type AuthTokenPayload = {
  token: string;
  user: AuthUser;
};

export const chatclawApi = {
  meta: () => apiFetch<MetaResponse>("/api/v1/meta"),

  auth: {
    register: (body: { email: string; password: string }) =>
      apiFetch<AuthTokenPayload>("/api/v1/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    login: (body: { email: string; password: string }) =>
      apiFetch<AuthTokenPayload>("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    me: () => apiFetch<{ user: AuthUser }>("/api/v1/auth/me"),
    patchMe: (body: { email: string }) =>
      apiFetch<{ user: AuthUser }>("/api/v1/auth/me", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },

  projects: {
    list: () => apiFetch<Project[]>("/api/v1/projects"),
    get: (id: string) => apiFetch<Project>(`/api/v1/projects/${id}`),
    create: (body: { name: string; description?: string; category?: string }) =>
      apiFetch<Project>("/api/v1/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    patch: (id: string, body: ProjectPatchBody) =>
      apiFetch<Project>(`/api/v1/projects/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    remove: (id: string) =>
      apiFetch<{ ok: boolean }>(`/api/v1/projects/${id}`, { method: "DELETE" }),
    materials: {
      list: (projectId: string) =>
        apiFetch<StudioMaterial[]>(`/api/v1/projects/${projectId}/materials`),
      get: (projectId: string, materialId: number) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/${materialId}`),
      create: (projectId: string, body: CreateStudioMaterialBody) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      patch: (projectId: string, materialId: number, body: PatchStudioMaterialBody) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/${materialId}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      /** Build .pptx from Markdown outline (server packs real OOXML; Keynote/PowerPoint compatible). */
      createSlidesPptx: (
        projectId: string,
        body: {
          title: string;
          markdown: string;
          language?: string;
          conversation_id?: number;
          material_id?: number;
        },
      ) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/slides-pptx`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      /** Standalone HTML page from AI body fragment or full document (server writes file). */
      createStudioHtml: (
        projectId: string,
        body: { title: string; markdown: string; language?: string; material_id?: number },
      ) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/studio-html`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      /** Markmap standalone HTML from nested Markdown list (server writes file). */
      createStudioMindmap: (
        projectId: string,
        body: { title: string; markdown: string; language?: string; material_id?: number },
      ) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/studio-mindmap`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      /** Binary audio (e.g. MP3 from /api/tts) stored under project materials; preview via studio-file. */
      createStudioAudio: (
        projectId: string,
        body: {
          title: string;
          base64_data: string;
          mime_type?: string;
          transcript_markdown?: string;
          material_id?: number;
        },
      ) =>
        apiFetch<StudioMaterial>(`/api/v1/projects/${projectId}/materials/studio-audio`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),

      /** V2: Unified studio create endpoint - backend handles prompt building and routing */
      studioCreate: (
        projectId: string,
        body: {
          type: "html" | "ppt" | "audio" | "mindmap" | "infographic" | "quiz" | "data_table";
          content: string;
          title?: string;
          options?: Record<string, unknown>;
          material_id?: number;
        },
      ) =>
        apiFetch<{
          success: boolean;
          material_id?: number;
          status: string;
          job_id?: string;
          error?: string;
          result?: Record<string, unknown>;
        }>(`/api/v1/projects/${projectId}/studio/create`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),

      /** V2: List available studio skills */
      studioSkills: () =>
        apiFetch<
          {
            type: string;
            name: string;
            version: string;
            description: string;
            trigger_keywords: string[];
            input_schema: { required: string[]; optional?: string[] };
            output_schema: { fields: string[] };
            theme_options?: Record<string, unknown>;
            voice_options?: Record<string, unknown>;
            limits?: Record<string, unknown>;
          }[]
        >("/api/v1/studio/skills"),
    },
  },

  agents: {
    list: () => apiFetch<Agent[]>("/api/v1/agents"),
    get: (id: number) => apiFetch<Agent>(`/api/v1/agents/${id}`),
    create: (body: { name: string; prompt: string; icon: string }) =>
      apiFetch<Agent>("/api/v1/agents", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    defaultPrompt: () => apiFetch<{ prompt: string }>("/api/v1/agents/default-prompt"),
    patch: (id: number, body: Record<string, unknown>) =>
      apiFetch<Agent>(`/api/v1/agents/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    remove: (id: number) =>
      apiFetch<{ ok: boolean }>(`/api/v1/agents/${id}`, { method: "DELETE" }),
  },

  libraries: {
    list: () => apiFetch<Library[]>("/api/v1/libraries"),
    create: (body: { name: string }) =>
      apiFetch<Library>("/api/v1/libraries", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },

  documents: {
    query: (libraryId: number, body: Record<string, unknown>) =>
      apiFetch<unknown[]>(`/api/v1/libraries/${libraryId}/documents/query`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    uploadBrowser: (
      libraryId: number,
      files: { file_name: string; base64_data: string }[],
      folderId?: number | null,
    ) =>
      apiFetch<unknown[]>(`/api/v1/libraries/${libraryId}/documents/upload-browser`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          library_id: libraryId,
          files,
          folder_id: folderId ?? undefined,
        }),
      }),
    /** Server fetches URL (http/https only, size limit); stores response body as a library document. */
    importUrl: (libraryId: number, body: { url: string; title?: string }) =>
      apiFetch<unknown[]>(`/api/v1/libraries/${libraryId}/documents/import-url`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    remove: (id: number) =>
      apiFetch<{ ok: boolean }>(`/api/v1/documents/${id}`, { method: "DELETE" }),
    patch: (id: number, body: { original_name?: string; starred?: boolean }) =>
      apiFetch<LibraryDocumentRecord>(`/api/v1/documents/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    /** Debug: list document_nodes + doc_vec presence and short embedding preview */
    vectorInspect: (documentId: number) =>
      apiFetch<VectorInspectResult>(`/api/v1/documents/${documentId}/vector-inspect`),
    /** Base64 + metadata for attaching a library document to chat (same size limits as upload). */
    chatAttachment: (documentId: number) =>
      apiFetch<DocumentChatAttachment>(`/api/v1/documents/${documentId}/chat-attachment`),
  },

  conversations: {
    list: (agentId: number, agentType = "eino") =>
      apiFetch<Conversation[]>(
        `/api/v1/conversations?agent_id=${agentId}&agent_type=${encodeURIComponent(agentType)}`,
      ),
    create: (body: ConversationCreateBody) =>
      apiFetch<Conversation>("/api/v1/conversations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    /** Per agent + library: one hidden studio_only conversation for /chat/messages (Studio). */
    ensureStudio: (body: { agent_id: number; library_ids: number[]; chat_mode?: string }) =>
      apiFetch<Conversation>("/api/v1/conversations/ensure-studio", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    messages: async (conversationId: number) => {
      const raw = await apiFetch<Message[] | null>(`/api/v1/conversations/${conversationId}/messages`);
      return Array.isArray(raw) ? raw : [];
    },
    ensureThread: (conversationId: number) =>
      apiFetch<{ thread_id: string }>(`/api/v1/conversations/${conversationId}/ensure-thread`, {
        method: "POST",
      }),
    /** True when LangGraph thread lists a downloadable skill .pptx (poll after studio slides stream ends). */
    studioSlidesArtifactStatus: (conversationId: number) =>
      apiFetch<{ ready: boolean; artifact_path?: string }>(
        `/api/v1/conversations/${conversationId}/studio/slides-artifact-status`,
      ),
    patch: (conversationId: number, body: ConversationPatchBody) =>
      apiFetch<Conversation>(`/api/v1/conversations/${conversationId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    remove: (conversationId: number) =>
      apiFetch<{ ok: boolean }>(`/api/v1/conversations/${conversationId}`, { method: "DELETE" }),
  },

  settings: {
    list: (category?: string) =>
      apiFetch<unknown[]>(
        `/api/v1/settings${category ? `?category=${encodeURIComponent(category)}` : ""}`,
      ),
    set: (key: string, value: string) =>
      apiFetch<unknown>("/api/v1/settings/set", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, value }),
      }),
    embedding: {
      get: () =>
        apiFetch<{ provider_id: string; model_id: string; dimension: number }>(
          "/api/v1/settings/embedding",
        ),
      put: (body: { provider_id: string; model_id: string; dimension: number }) =>
        apiFetch<{ ok: boolean }>("/api/v1/settings/embedding", {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
    },
  },

  providers: {
    list: () => apiFetch<ProviderDTO[]>("/api/v1/providers"),
    get: (id: string) => apiFetch<ProviderDTO>(`/api/v1/providers/${encodeURIComponent(id)}`),
    patch: (id: string, body: ProviderPatchBody) =>
      apiFetch<ProviderDTO>(`/api/v1/providers/${encodeURIComponent(id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    withModels: (id: string) => apiFetch<ProviderWithModelsDTO>(`/api/v1/providers/${encodeURIComponent(id)}/models`),
    checkApiKey: (id: string, body: { api_key?: string; api_endpoint?: string; extra_config?: string }) =>
      apiFetch<{ success: boolean; message: string }>(
        `/api/v1/providers/${encodeURIComponent(id)}/check-api-key`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    syncModels: (id: string) =>
      apiFetch<{ ok: boolean }>(`/api/v1/providers/${encodeURIComponent(id)}/sync-models`, {
        method: "POST",
      }),
    patchModel: (
      providerId: string,
      body: { model_id: string; enabled?: boolean; name?: string },
    ) =>
      apiFetch<ModelDTO>(`/api/v1/providers/${encodeURIComponent(providerId)}/models`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },

  skills: {
    installed: () => apiFetch<InstalledSkill[]>("/api/v1/skills/installed"),
    workspace: () => apiFetch<{ skills_dir: string }>("/api/v1/skills/workspace"),
    refresh: () =>
      apiFetch<InstalledSkill[]>("/api/v1/skills/refresh", { method: "POST" }),
    install: (slug: string, version: string) =>
      apiFetch<{ ok: boolean }>("/api/v1/skills/install", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slug, version }),
      }),
    enable: (slug: string) =>
      apiFetch<{ ok: boolean }>(`/api/v1/skills/${encodeURIComponent(slug)}/enable`, {
        method: "POST",
      }),
    disable: (slug: string) =>
      apiFetch<{ ok: boolean }>(`/api/v1/skills/${encodeURIComponent(slug)}/disable`, {
        method: "POST",
      }),
    uninstall: (slug: string) =>
      apiFetch<{ ok: boolean }>(`/api/v1/skills/${encodeURIComponent(slug)}`, { method: "DELETE" }),
  },

  mcp: {
    servers: () => apiFetch<unknown[]>("/api/v1/mcp/servers"),
  },

  toolchain: {
    status: () => apiFetch<unknown[]>("/api/v1/toolchain/status"),
  },

  scheduled: {
    list: () => apiFetch<unknown[]>("/api/v1/scheduled-tasks"),
  },

  channels: {
    list: () => apiFetch<unknown[]>("/api/v1/channels"),
    platforms: () => apiFetch<unknown[]>("/api/v1/channels/platforms"),
  },

  memory: {
    files: (openclawAgentId: string) =>
      apiFetch<unknown[]>(
        `/api/v1/memory/files?openclaw_agent_id=${encodeURIComponent(openclawAgentId)}`,
      ),
  },
};
