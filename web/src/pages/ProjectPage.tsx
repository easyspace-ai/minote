import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search, SquarePen } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Navigate, useLocation, useParams } from "react-router-dom";
import { toast } from "sonner";
import {
  chatclawApi,
  type Agent,
  type Conversation,
  type StudioMaterial,
  type StudioMaterialKind,
  type StudioScopeSettings,
} from "@/api/chatclaw";
import { NotebookShell, type NotebookWorkbenchTab } from "@/components/layout/NotebookShell";
import { ArtifactFileDetail, ArtifactsProvider } from "@/components/workspace/artifacts";
import { ProjectAddSourceDialog } from "@/components/project/ProjectAddSourceDialog";
import { LibraryDocumentPreviewPane } from "@/components/project/LibraryDocumentPreview";
import { ProjectConversationRow } from "@/components/project/ProjectConversationRow";
import { ProjectLibraryDocRow } from "@/components/project/ProjectLibraryDocRow";
import { ProjectOverflowMenu } from "@/components/project/ProjectOverflowMenu";
import { ChatWorkspaceView } from "@/pages/workspace/chats/ChatWorkspaceView";
import { StudioMaterialExpandedOverlay, StudioMaterialPreviewPane } from "@/components/workspace/StudioMaterialDialog";
import { StudioPanel } from "@/components/workspace/StudioPanel";
import { isLibraryDocumentProcessing, type LibraryDocumentRow as LibraryDoc } from "@/lib/libraryDocumentStatus";
import { useChatSend } from "@/hooks/useChatSend";
import {
  buildStudioChatExcerpt,
  buildStudioGenerationPrompt,
  buildStudioScopePrefix,
  stripMarkdownFences,
  STUDIO_CHAT_EXCERPT_MAX_RUNES,
  truncateToRunes,
} from "@/lib/studioPrompts";
import { arrayBufferToBase64, synthesizePodcastSpeechMp3 } from "@/lib/studioPodcastTts";
import { downloadLibraryDocument } from "@/lib/libraryDocumentDownload";

function ProjectDeerFlowChat({
  convId,
  seedUserMessage,
  seedUserFiles,
  seedChatMode,
  studioDocumentIds = [],
  onChatThreadId,
}: {
  convId: number | null;
  seedUserMessage?: string | null;
  seedUserFiles?: File[] | undefined;
  seedChatMode?: "flash" | "thinking" | "pro" | "ultra";
  /** Checked library docs → same full-text injection as Studio (LangGraph run context). */
  studioDocumentIds?: number[];
  onChatThreadId?: (threadId: string | null) => void;
}) {
  const [threadId, setThreadId] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  const streamExtraContext = useMemo(() => {
    if (convId == null) return undefined;
    const ctx: Record<string, unknown> = { conversation_id: convId };
    if (studioDocumentIds.length > 0) {
      ctx.studio_document_ids = studioDocumentIds;
    }
    return ctx;
  }, [convId, studioDocumentIds]);

  useEffect(() => {
    if (convId == null) {
      setThreadId(null);
      setReady(false);
      onChatThreadId?.(null);
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const { thread_id } = await chatclawApi.conversations.ensureThread(convId);
        if (!cancelled) {
          setThreadId(thread_id);
          setReady(true);
        }
      } catch {
        if (!cancelled) {
          setThreadId(null);
          setReady(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [convId, onChatThreadId]);

  useEffect(() => {
    if (convId != null && threadId) {
      onChatThreadId?.(threadId);
    }
  }, [convId, threadId, onChatThreadId]);

  if (convId == null || !ready || !threadId) {
    return (
      <div className="text-muted-foreground flex h-full min-h-[320px] items-center justify-center text-sm">
        准备会话…
      </div>
    );
  }

  return (
    <ChatWorkspaceView
      threadId={threadId}
      isNewThread={false}
      setIsNewThread={() => {}}
      replaceUrlOnStart={false}
      seedUserMessage={seedUserMessage}
      seedUserFiles={seedUserFiles}
      seedChatMode={seedChatMode}
      streamExtraContext={streamExtraContext}
      skipArtifactsProvider
    />
  );
}

export function ProjectPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const location = useLocation();
  const id = Number(projectId);
  const qc = useQueryClient();
  const ensureKeyRef = useRef<string | null>(null);
  const [firstConvId, setFirstConvId] = useState<number | null>(null);

  type ProjectNavState = {
    initialPrompt?: string;
    initialFiles?: unknown;
    initialChatMode?: "flash" | "thinking" | "pro" | "ultra";
  };

  const [landingSeed] = useState(() => {
    const st = location.state as ProjectNavState | null | undefined;
    const prompt = typeof st?.initialPrompt === "string" ? st.initialPrompt : null;
    const raw = st?.initialFiles;
    const files = Array.isArray(raw)
      ? raw.filter((f): f is File => f instanceof File)
      : [];
    const chatMode = st?.initialChatMode;
    if (!prompt?.trim() && files.length === 0) return null;
    return { prompt, files, chatMode };
  });

  const [agentId, setAgentId] = useState<number | null>(null);
  const [convId, setConvId] = useState<number | null>(null);
  const [libraryPreviewDoc, setLibraryPreviewDoc] = useState<LibraryDoc | null>(null);
  /** 右栏内联替换 Studio 列表（NotebookLM 式，非抽屉） */
  const [studioSidebarMaterial, setStudioSidebarMaterial] = useState<StudioMaterial | null>(null);
  const [studioExpandOpen, setStudioExpandOpen] = useState(false);
  const [studioErr, setStudioErr] = useState<string | null>(null);
  const [docSearchQuery, setDocSearchQuery] = useState("");
  const [docSearchOpen, setDocSearchOpen] = useState(false);
  const [convSearchQuery, setConvSearchQuery] = useState("");
  const [convSearchOpen, setConvSearchOpen] = useState(false);
  const [addSourceOpen, setAddSourceOpen] = useState(false);
  const [workbenchTab, setWorkbenchTab] = useState<NotebookWorkbenchTab>("对话");
  const [chatThreadId, setChatThreadId] = useState<string | null>(null);
  const [chatArtifactPath, setChatArtifactPath] = useState<string | null>(null);
  /** Studio: per-document inclusion for full-text injection (document_id → checked). */
  const [studioDocPick, setStudioDocPick] = useState<Map<number, boolean>>(() => new Map());
  const [studioIncludeChat, setStudioIncludeChat] = useState(false);
  const [studioChatSummaryOnly, setStudioChatSummaryOnly] = useState(false);
  const [studioChatMaxMessages, setStudioChatMaxMessages] = useState("");
  const [studioCustomExtra, setStudioCustomExtra] = useState("");

  const handleWorkbenchTabChange = useCallback((tab: NotebookWorkbenchTab) => {
    setWorkbenchTab(tab);
    if (tab === "对话") {
      setLibraryPreviewDoc(null);
      setStudioSidebarMaterial(null);
      setStudioExpandOpen(false);
      setChatArtifactPath(null);
    }
  }, []);

  useEffect(() => {
    if (!studioSidebarMaterial || studioExpandOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.preventDefault();
      setStudioSidebarMaterial(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [studioSidebarMaterial, studioExpandOpen]);

  const project = useQuery({
    queryKey: ["project", id],
    queryFn: () => chatclawApi.projects.get(id),
    enabled: Number.isFinite(id) && id > 0,
  });

  const agents = useQuery({
    queryKey: ["agents"],
    queryFn: () => chatclawApi.agents.list(),
  });

  useEffect(() => {
    if (!agents.data?.length) return;
    if (agentId == null || !agents.data.some((a) => a.id === agentId)) {
      setAgentId(agents.data[0]!.id);
    }
  }, [agents.data, agentId]);

  const libraryId = project.data?.library_id;

  const conversations = useQuery({
    queryKey: ["conversations", agentId],
    queryFn: () => chatclawApi.conversations.list(agentId!),
    enabled: agentId != null && agentId > 0,
  });

  const materials = useQuery({
    queryKey: ["project-materials", id],
    queryFn: () => chatclawApi.projects.materials.list(id),
    enabled: Number.isFinite(id) && id > 0,
  });

  const docs = useQuery({
    queryKey: ["documents", libraryId],
    queryFn: () =>
      chatclawApi.documents.query(libraryId!, {
        limit: 80,
        sort_by: "created_desc",
        folder_id: 0,
        keyword: "",
        before_id: 0,
      }),
    enabled: libraryId != null && libraryId > 0,
    refetchInterval: (q) => {
      const rows = q.state.data;
      if (!Array.isArray(rows) || rows.length === 0) return false;
      return (rows as LibraryDoc[]).some(isLibraryDocumentProcessing) ? 1200 : false;
    },
  });

  const docRows = useMemo((): LibraryDoc[] => {
    const raw = docs.data;
    return Array.isArray(raw) ? (raw as LibraryDoc[]) : [];
  }, [docs.data]);

  const studioChatDocIds = useMemo(
    () => docRows.filter((d) => studioDocPick.get(d.id) !== false).map((d) => d.id),
    [docRows, studioDocPick],
  );

  useEffect(() => {
    setStudioDocPick((prev) => {
      const next = new Map<number, boolean>();
      for (const d of docRows) {
        next.set(d.id, prev.has(d.id) ? (prev.get(d.id) ?? true) : true);
      }
      return next;
    });
  }, [docRows]);

  useEffect(() => {
    const p = project.data;
    if (!p) return;
    const s = p.studio_scope;
    setStudioIncludeChat(Boolean(s?.includeChat));
    setStudioChatSummaryOnly(Boolean(s?.chatSummaryOnly));
    setStudioChatMaxMessages(typeof s?.chatMaxMessages === "string" ? s.chatMaxMessages : "");
    setStudioCustomExtra(typeof s?.customExtra === "string" ? s.customExtra : "");
  }, [project.data?.id, project.data?.updated_at]);

  const { sendStream } = useChatSend();

  const saveStudioScopeMutation = useMutation({
    mutationFn: (scope: StudioScopeSettings) =>
      chatclawApi.projects.patch(id, { studio_scope: scope }),
    onSuccess: (updated) => {
      toast.success("已保存生成范围设置");
      qc.setQueryData(["project", id], updated);
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "保存失败");
    },
  });

  useEffect(() => {
    ensureKeyRef.current = null;
    setConvId(null);
  }, [id, agentId]);

  useEffect(() => {
    setFirstConvId(null);
  }, [id]);

  useEffect(() => {
    setChatThreadId(null);
    setChatArtifactPath(null);
  }, [convId]);

  useEffect(() => {
    if (convId != null && firstConvId == null) {
      setFirstConvId(convId);
    }
  }, [convId, firstConvId]);

  const seedForActiveChat =
    landingSeed && convId != null && convId === firstConvId ? landingSeed : null;

  useEffect(() => {
    if (!project.data || !agentId || !conversations.isSuccess || libraryId == null) return;
    const libId = libraryId;
    const list = Array.isArray(conversations.data) ? conversations.data : [];
    const match = list.filter((c: Conversation) => c.library_ids?.includes(libId));

    if (convId != null && match.some((c) => c.id === convId)) {
      return;
    }
    if (match.length > 0) {
      setConvId(match[0]!.id);
      return;
    }
    const k = `${id}-${agentId}`;
    if (ensureKeyRef.current === k) return;
    ensureKeyRef.current = k;
    void (async () => {
      try {
        const c = await chatclawApi.conversations.create({
          agent_id: agentId,
          name: `${project.data.name} · 讨论`,
          chat_mode: "chat",
          library_ids: [libId],
        });
        setConvId(c.id);
        void qc.invalidateQueries({ queryKey: ["conversations", agentId] });
      } catch {
        ensureKeyRef.current = null;
      }
    })();
  }, [project.data, agentId, conversations.isSuccess, conversations.data, libraryId, id, qc, convId]);

  const createConversation = useMutation({
    mutationFn: async () => {
      if (agentId == null || libraryId == null) {
        throw new Error("项目或助手未就绪");
      }
      const label = new Date().toLocaleString("zh-CN", {
        month: "numeric",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
      return chatclawApi.conversations.create({
        agent_id: agentId,
        name: `新会话 · ${label}`,
        chat_mode: "chat",
        library_ids: [libraryId],
      });
    },
    onSuccess: (c) => {
      setConvId(c.id);
      void qc.invalidateQueries({ queryKey: ["conversations", agentId] });
    },
  });

  const STUDIO_RAG_KINDS: ReadonlySet<StudioMaterialKind> = new Set([
    "audio",
    "slides",
    "html",
    "mindmap",
  ]);

  const studioGenerate = useMutation({
    mutationFn: async ({
      kind,
      title,
      selectedDocIds,
      includeChat,
      chatSummaryOnly,
      chatMaxMessages,
      customExtra,
    }: {
      kind: StudioMaterialKind;
      title: string;
      selectedDocIds: number[];
      includeChat: boolean;
      chatSummaryOnly: boolean;
      chatMaxMessages: number;
      customExtra: string;
    }) => {
      if (selectedDocIds.length === 0) {
        throw new Error("请至少勾选一份资料用于 Studio 生成。");
      }
      if (!STUDIO_RAG_KINDS.has(kind)) {
        throw new Error(
          "该类型请先在对话中说明需求；Studio 快捷入口当前支持：音频概述、幻灯片、网页、思维导图。",
        );
      }
      if (convId == null) {
        throw new Error("会话未就绪，请稍候再试。");
      }

      const displayTitle = title.trim() || "未命名";
      const selectedDocs = docRows.filter((d) => selectedDocIds.includes(d.id));

      let chatExcerpt = "";
      if (includeChat) {
        try {
          const msgs = await chatclawApi.conversations.messages(convId);
          chatExcerpt = truncateToRunes(
            buildStudioChatExcerpt(msgs, { maxMessages: chatMaxMessages, summaryOnly: chatSummaryOnly }),
            STUDIO_CHAT_EXCERPT_MAX_RUNES,
          );
        } catch {
          chatExcerpt = "";
        }
      }

      const scopePrefix = buildStudioScopePrefix({
        selectedDocuments: selectedDocs.map((d) => ({ id: d.id, name: d.original_name })),
        chatExcerpt: chatExcerpt || undefined,
        customInstruction: customExtra.trim() || undefined,
      });
      const userContent = scopePrefix + buildStudioGenerationPrompt(kind, displayTitle);

      const pending = await chatclawApi.projects.materials.create(id, {
        kind,
        title: displayTitle,
        status: "pending",
        subtitle: "生成中…",
        payload: {},
      });
      const materialId = pending.id;
      void qc.invalidateQueries({ queryKey: ["project-materials", id] });

      let assistantText = "";
      try {
        await sendStream(
          convId,
          userContent,
          "project-studio",
          {
            onChunk: (d) => {
              assistantText += d;
            },
          },
          { studioDocumentIds: selectedDocIds },
        );

        const raw = stripMarkdownFences(assistantText).trim();
        if (kind !== "slides" && !raw) {
          throw new Error("模型未返回可用内容，请重试或调整选用资料。");
        }

        switch (kind) {
          case "slides": {
            const mdForPayload =
              raw ||
              `# ${displayTitle}\n\n_正文与版式以 presentation skill 生成的 .pptx 为准；本条仅作占位。_`;
            await chatclawApi.projects.materials.createSlidesPptx(id, {
              title: displayTitle,
              markdown: mdForPayload,
              conversation_id: convId,
              material_id: materialId,
            });
            break;
          }
          case "html":
            await chatclawApi.projects.materials.createStudioHtml(id, {
              title: displayTitle,
              markdown: raw,
              material_id: materialId,
            });
            break;
          case "mindmap":
            await chatclawApi.projects.materials.createStudioMindmap(id, {
              title: displayTitle,
              markdown: raw,
              material_id: materialId,
            });
            break;
          case "audio": {
            const mp3 = await synthesizePodcastSpeechMp3(raw);
            await chatclawApi.projects.materials.createStudioAudio(id, {
              title: displayTitle,
              base64_data: arrayBufferToBase64(mp3),
              mime_type: "audio/mpeg",
              transcript_markdown: raw,
              material_id: materialId,
            });
            break;
          }
          default:
            throw new Error("不支持的 Studio 类型");
        }
      } catch (err) {
        try {
          await chatclawApi.projects.materials.patch(id, materialId, {
            status: "failed",
            subtitle: "生成失败",
          });
        } catch {
          /* ignore */
        }
        throw err;
      }

      return { materialId };
    },
    onSuccess: () => {
      setStudioErr(null);
      void qc.invalidateQueries({ queryKey: ["project-materials", id] });
    },
    onError: (e) => {
      setStudioErr(e instanceof Error ? e.message : "生成失败");
      void qc.invalidateQueries({ queryKey: ["project-materials", id] });
    },
  });

  const deleteLibraryDocument = useMutation({
    mutationFn: (documentId: number) => chatclawApi.documents.remove(documentId),
    onSuccess: (_, documentId) => {
      toast.success("已删除");
      const libId = project.data?.library_id;
      if (libId != null) {
        void qc.invalidateQueries({ queryKey: ["documents", libId] });
      }
      setStudioDocPick((prev) => {
        const next = new Map(prev);
        next.delete(documentId);
        return next;
      });
      setLibraryPreviewDoc((d) => (d?.id === documentId ? null : d));
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "删除失败");
    },
  });

  const patchLibraryDocument = useMutation({
    mutationFn: ({
      documentId,
      body,
    }: {
      documentId: number;
      body: { original_name?: string; starred?: boolean };
    }) => chatclawApi.documents.patch(documentId, body),
    onSuccess: (data, { documentId }) => {
      toast.success("已更新");
      const libId = project.data?.library_id;
      if (libId != null) {
        void qc.invalidateQueries({ queryKey: ["documents", libId] });
      }
      setLibraryPreviewDoc((prev) => {
        if (prev?.id !== documentId) return prev;
        return {
          ...prev,
          original_name: data.original_name,
          starred: data.starred ?? prev.starred,
        };
      });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "更新失败");
    },
  });

  const patchConversationMutation = useMutation({
    mutationFn: ({ conversationId, name }: { conversationId: number; name: string }) =>
      chatclawApi.conversations.patch(conversationId, { name }),
    onSuccess: () => {
      toast.success("已重命名");
      if (agentId != null) {
        void qc.invalidateQueries({ queryKey: ["conversations", agentId] });
      }
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "重命名失败");
    },
  });

  const deleteConversationMutation = useMutation({
    mutationFn: (conversationId: number) => chatclawApi.conversations.remove(conversationId),
    onSuccess: (_, deletedId) => {
      toast.success("已删除");
      const aid = agentId;
      if (aid == null) {
        return;
      }
      qc.setQueryData<Conversation[]>(["conversations", aid], (old) =>
        (old ?? []).filter((c) => c.id !== deletedId),
      );
      const wasActive = convId === deletedId;
      if (wasActive) {
        const list = qc.getQueryData<Conversation[]>(["conversations", aid]) ?? [];
        const libId = project.data?.library_id;
        const forLib =
          libId != null ? list.filter((c) => c.library_ids?.includes(libId)) : list;
        setConvId(forLib[0]?.id ?? null);
      }
      void qc.invalidateQueries({ queryKey: ["conversations", aid] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "删除失败");
    },
  });

  const handleDownloadLibraryDoc = useCallback(async (d: LibraryDoc) => {
    try {
      await downloadLibraryDocument(d.id, d.original_name);
      toast.success("已开始下载");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "下载失败");
    }
  }, []);

  if (!Number.isFinite(id) || id <= 0) {
    return <Navigate to="/" replace />;
  }
  if (project.isError) {
    return <Navigate to="/" replace />;
  }
  if (project.isLoading || !project.data) {
    return (
      <div className="flex h-full items-center justify-center bg-white text-muted-foreground">
        加载项目…
      </div>
    );
  }

  const p = project.data;

  const libraryNotReady = libraryId == null || libraryId <= 0;

  const docQ = docSearchQuery.trim().toLowerCase();
  const filteredDocs = docQ
    ? docRows.filter((d) => d.original_name.toLowerCase().includes(docQ))
    : docRows;

  const convRows = (Array.isArray(conversations.data) ? conversations.data : []).filter((c: Conversation) =>
    libraryId != null ? (c.library_ids?.includes(libraryId) ?? false) : false,
  );
  const convQ = convSearchQuery.trim().toLowerCase();
  const filteredConvs = convQ
    ? convRows.filter(
        (c) =>
          (c.name || "").toLowerCase().includes(convQ) ||
          (c.last_message || "").toLowerCase().includes(convQ),
      )
    : convRows;

  const openMaterial = (m: StudioMaterial) => {
    setLibraryPreviewDoc(null);
    setChatArtifactPath(null);
    setStudioSidebarMaterial(m);
    setStudioExpandOpen(false);
  };

  /** 资料列表（图三：紧凑行 + 圆角，无大卡片） */
  const materialsDocList = (
    <ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto pr-0.5">
      {docs.isLoading && <li className="text-muted-foreground px-1 py-2 text-xs">加载文档…</li>}
      {filteredDocs.map((d) => (
        <li key={d.id}>
          <ProjectLibraryDocRow
            doc={d}
            onPreview={() => setLibraryPreviewDoc(d)}
            onDownload={() => void handleDownloadLibraryDoc(d)}
            onAddMaterial={() => setAddSourceOpen(true)}
            onDelete={() => {
              if (!window.confirm(`确定将「${d.original_name}」移到回收站？`)) return;
              deleteLibraryDocument.mutate(d.id);
            }}
            onRename={() => {
              const next = window.prompt("新的文件名", d.original_name);
              if (next == null) return;
              const t = next.trim();
              if (!t || t === d.original_name) return;
              patchLibraryDocument.mutate({ documentId: d.id, body: { original_name: t } });
            }}
            onStarredChange={(starred) => {
              if (starred === Boolean(d.starred)) return;
              patchLibraryDocument.mutate({ documentId: d.id, body: { starred } });
            }}
            studioSelect={{
              checked: studioDocPick.get(d.id) !== false,
              onChange: (checked) => {
                setStudioDocPick((prev) => {
                  const next = new Map(prev);
                  next.set(d.id, checked);
                  return next;
                });
              },
            }}
          />
        </li>
      ))}
      {libraryId && docs.isFetched && docRows.length === 0 && (
        <li className="text-muted-foreground px-1 py-2 text-xs">暂无文档，点击右上角 + 添加来源。</li>
      )}
      {libraryId && docs.isFetched && docRows.length > 0 && filteredDocs.length === 0 && (
        <li className="text-muted-foreground px-1 py-2 text-xs">无匹配文档。</li>
      )}
    </ul>
  );

  const leftPanelMaterials = (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex shrink-0 items-center justify-between gap-2">
        <h2 className="text-foreground text-sm font-semibold tracking-tight">资料</h2>
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            type="button"
            title="搜索"
            onClick={() => setDocSearchOpen((v) => !v)}
            className="text-muted-foreground hover:text-foreground rounded-lg p-1.5 transition-colors hover:bg-[#E5E5E5]"
          >
            <Search className="h-4 w-4" />
          </button>
          <button
            type="button"
            title="添加来源"
            onClick={() => setAddSourceOpen(true)}
            disabled={libraryNotReady}
            className="text-muted-foreground hover:text-foreground rounded-lg p-1.5 transition-colors hover:bg-[#E5E5E5] disabled:pointer-events-none disabled:opacity-40"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
      {docSearchOpen ? (
        <input
          type="search"
          value={docSearchQuery}
          onChange={(e) => setDocSearchQuery(e.target.value)}
          placeholder="筛选文档…"
          className="border-border bg-background text-foreground placeholder:text-muted-foreground shrink-0 rounded-lg border px-2.5 py-1.5 text-xs focus:border-black/20 focus:outline-none focus:ring-2 focus:ring-black/10"
        />
      ) : null}
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <p className="text-muted-foreground min-w-0 flex-1 text-[11px] leading-snug">
          勾选资料将全文注入 Studio 与当前对话；未勾选的资料不会参与回答。
        </p>
        <button
          type="button"
          className="text-foreground/80 hover:bg-black/[0.06] shrink-0 rounded-lg px-2 py-1 text-[11px] font-medium"
          onClick={() => setStudioDocPick(new Map(docRows.map((d) => [d.id, true])))}
        >
          全选
        </button>
        <button
          type="button"
          className="text-foreground/80 hover:bg-black/[0.06] shrink-0 rounded-lg px-2 py-1 text-[11px] font-medium"
          onClick={() => setStudioDocPick(new Map(docRows.map((d) => [d.id, false])))}
        >
          全不选
        </button>
      </div>
      {materialsDocList}
    </div>
  );

  const leftPanelConversations = (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex shrink-0 items-center justify-between gap-2">
        <h2 className="text-foreground text-sm font-semibold tracking-tight">对话</h2>
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            type="button"
            title="搜索会话"
            onClick={() => setConvSearchOpen((v) => !v)}
            className="text-muted-foreground hover:text-foreground rounded-lg p-1.5 transition-colors hover:bg-[#E5E5E5]"
          >
            <Search className="h-4 w-4" />
          </button>
          <button
            type="button"
            title="新会话"
            disabled={agentId == null || libraryId == null || createConversation.isPending}
            onClick={() => createConversation.mutate()}
            className="text-muted-foreground hover:text-foreground rounded-lg p-1.5 transition-colors hover:bg-[#E5E5E5] disabled:pointer-events-none disabled:opacity-40"
          >
            <SquarePen className="h-4 w-4" />
          </button>
        </div>
      </div>
      {convSearchOpen ? (
        <input
          type="search"
          value={convSearchQuery}
          onChange={(e) => setConvSearchQuery(e.target.value)}
          placeholder="筛选会话…"
          className="border-border bg-background text-foreground placeholder:text-muted-foreground shrink-0 rounded-lg border px-2.5 py-1.5 text-xs focus:border-black/20 focus:outline-none focus:ring-2 focus:ring-black/10"
        />
      ) : null}
      <ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto pr-0.5">
        {conversations.isLoading && <li className="text-muted-foreground px-1 py-2 text-xs">加载会话…</li>}
        {filteredConvs.map((c: Conversation) => (
          <li key={c.id}>
            <ProjectConversationRow
              conversation={c}
              active={convId === c.id}
              onSelect={() => setConvId(c.id)}
              onRename={() => {
                const label = c.name || `会话 #${c.id}`;
                const next = window.prompt("会话名称", label);
                if (next == null) return;
                const t = next.trim();
                if (!t || t === label) return;
                patchConversationMutation.mutate({ conversationId: c.id, name: t });
              }}
              onDelete={() => {
                const label = c.name || `会话 #${c.id}`;
                if (!window.confirm(`确定删除「${label}」？此操作不可恢复。`)) return;
                deleteConversationMutation.mutate(c.id);
              }}
            />
          </li>
        ))}
        {!conversations.isLoading && convRows.length === 0 && (
          <li className="text-muted-foreground px-1 py-2 text-xs">暂无会话，将自动创建或点此栏「新会话」。</li>
        )}
        {!conversations.isLoading && convRows.length > 0 && filteredConvs.length === 0 && (
          <li className="text-muted-foreground px-1 py-2 text-xs">无匹配会话。</li>
        )}
      </ul>
    </div>
  );

  const mats = materials.data ?? [];
  const leftPanelResults = (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <h2 className="text-foreground shrink-0 text-sm font-semibold tracking-tight">结果</h2>
      <p className="text-muted-foreground shrink-0 text-[11px] leading-snug">生成物摘要；点击在右侧 Studio 查看。</p>
      <ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto pr-0.5">
        {materials.isLoading && <li className="text-muted-foreground px-1 py-2 text-xs">加载…</li>}
        {!materials.isLoading && mats.length === 0 && (
          <li className="text-muted-foreground px-1 py-2 text-xs">暂无结果，可在右侧工作室生成。</li>
        )}
        {mats.map((m) => (
          <li key={m.id}>
            <button
              type="button"
              onClick={() => openMaterial(m)}
              className="w-full rounded-xl px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-[#E5E5E5]"
            >
              <span className="truncate font-medium">{m.title}</span>
              <span className="text-muted-foreground mt-0.5 block text-[10px]">{m.kind}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );

  const leftByTab: Record<NotebookWorkbenchTab, ReactNode> = {
    资料: leftPanelMaterials,
    对话: leftPanelConversations,
   
  };

  const studioColumn = (
    <div className="flex h-full min-h-0 min-w-0 flex-col gap-2">
            {studioErr ? (
              <div className="shrink-0 rounded-xl border border-outline-variant/30 bg-popover px-3 py-2 text-xs text-popover-foreground shadow-sm dark:shadow-none dark:ring-1 dark:ring-white/10">
                {studioErr}
              </div>
            ) : null}
            {studioSidebarMaterial ? (
              <StudioMaterialPreviewPane
                variant="sidebar"
                material={studioSidebarMaterial}
                projectId={id}
                onClose={() => {
                  setStudioSidebarMaterial(null);
                  setStudioExpandOpen(false);
                }}
                onExpand={() => setStudioExpandOpen(true)}
              />
            ) : chatArtifactPath && chatThreadId ? (
              <ArtifactFileDetail
                className="min-h-0 flex-1"
                filepath={chatArtifactPath}
                threadId={chatThreadId}
                onCloseRequest={() => setChatArtifactPath(null)}
              />
            ) : (
            <StudioPanel
              materials={[...(materials.data ?? [])].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())}
              materialsLoading={materials.isLoading}
              onSelectMaterial={openMaterial}
              onSaveScopeSettings={() =>
                saveStudioScopeMutation.mutateAsync({
                  includeChat: studioIncludeChat,
                  chatSummaryOnly: studioChatSummaryOnly,
                  chatMaxMessages: studioChatMaxMessages,
                  customExtra: studioCustomExtra,
                })
              }
              scopeSavePending={saveStudioScopeMutation.isPending}
              scopeControls={
                <div className="space-y-3 text-foreground">
                  <label className="flex cursor-pointer items-start gap-2.5">
                    <input
                      type="checkbox"
                      className="mt-0.5 h-4 w-4 accent-black"
                      checked={studioIncludeChat}
                      onChange={(e) => setStudioIncludeChat(e.target.checked)}
                    />
                    <span>包含当前会话节选（附在用户提示中；与勾选资料一并作为依据）</span>
                  </label>
                  <label
                    className={`flex cursor-pointer items-start gap-2.5 pl-6 ${!studioIncludeChat ? "opacity-40" : ""}`}
                  >
                    <input
                      type="checkbox"
                      className="mt-0.5 h-4 w-4 accent-black"
                      disabled={!studioIncludeChat}
                      checked={studioChatSummaryOnly}
                      onChange={(e) => setStudioChatSummaryOnly(e.target.checked)}
                    />
                    <span>仅最近一条助手回复（作会话摘要）</span>
                  </label>
                  <div className={`flex flex-wrap items-center gap-2 pl-6 ${!studioIncludeChat || studioChatSummaryOnly ? "opacity-40" : ""}`}>
                    <label htmlFor="studio-chat-max" className="text-muted-foreground shrink-0">
                      会话消息条数（尾部）
                    </label>
                    <input
                      id="studio-chat-max"
                      type="number"
                      min={1}
                      max={200}
                      placeholder="默认 24"
                      disabled={!studioIncludeChat || studioChatSummaryOnly}
                      value={studioChatMaxMessages}
                      onChange={(e) => setStudioChatMaxMessages(e.target.value)}
                      className="border-border bg-background w-20 rounded-lg border px-2 py-1 text-xs"
                    />
                  </div>
                  <div>
                    <label htmlFor="studio-custom-extra" className="text-muted-foreground mb-1 block text-[11px]">
                      自定义补充（风格、受众、结构等）
                    </label>
                    <textarea
                      id="studio-custom-extra"
                      rows={3}
                      value={studioCustomExtra}
                      onChange={(e) => setStudioCustomExtra(e.target.value)}
                      placeholder="可选：例如「面向初学者」「突出安全注意事项」…"
                      className="border-border bg-background placeholder:text-muted-foreground w-full resize-y rounded-xl border px-2.5 py-2 text-xs"
                    />
                  </div>
                </div>
              }
              onQuickMaterial={(kind, title) => {
                setStudioErr(null);
                const selectedDocIds = docRows
                  .filter((d) => studioDocPick.get(d.id) !== false)
                  .map((d) => d.id);
                const parsedMax = parseInt(studioChatMaxMessages, 10);
                const chatMaxMessages =
                  Number.isFinite(parsedMax) && parsedMax >= 1 ? Math.min(200, parsedMax) : 24;
                studioGenerate.mutate({
                  kind,
                  title,
                  selectedDocIds,
                  includeChat: studioIncludeChat,
                  chatSummaryOnly: studioChatSummaryOnly,
                  chatMaxMessages,
                  customExtra: studioCustomExtra,
                });
              }}
              onSelectChatArtifact={(filepath) => {
                setStudioSidebarMaterial(null);
                setStudioExpandOpen(false);
                setChatArtifactPath(filepath);
              }}
            />
            )}
    </div>
  );

  const chatColumn = (
    <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-black/[0.06] px-6 py-3">
        <div className="flex min-w-0 flex-1 items-center gap-1">
          <div className="min-w-0 flex-1">
            <p className="text-xs font-medium text-muted-foreground">项目</p>
            <p className="truncate text-sm font-bold text-foreground">{p.name}</p>
          </div>
          <ProjectOverflowMenu project={p} compact />
        </div>
        {/* <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">助手</span>
          <select
            className="border-border/60 bg-background text-foreground max-w-[160px] rounded-lg border px-2 py-1 text-xs font-medium focus:border-foreground/25 focus:outline-none focus:ring-2 focus:ring-foreground/10"
            value={agentId ?? ""}
            onChange={(e) => setAgentId(Number(e.target.value))}
          >
            {agents.data?.map((a: Agent) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </div> */}
      </div>
      {libraryPreviewDoc ? (
        <div className="shrink-0 border-b border-black/[0.06] bg-white px-6 py-2 text-xs text-muted-foreground">
          <span className="font-medium text-foreground">当前资料</span>
          <span className="ml-2 truncate">{libraryPreviewDoc.original_name}</span>
        </div>
      ) : null}
      <ProjectDeerFlowChat
        key={convId ?? 0}
        convId={convId}
        seedUserMessage={seedForActiveChat?.prompt ?? null}
        seedUserFiles={seedForActiveChat?.files}
        seedChatMode={seedForActiveChat?.chatMode}
        studioDocumentIds={studioChatDocIds}
        onChatThreadId={setChatThreadId}
      />
    </div>
  );

  const shellCenter =
    libraryPreviewDoc != null ? (
      <LibraryDocumentPreviewPane
        doc={libraryPreviewDoc}
        embedded
        onClose={() => setLibraryPreviewDoc(null)}
      />
    ) : (
      chatColumn
    );

  const shellRight = libraryPreviewDoc != null ? chatColumn : studioColumn;

  return (
    <ArtifactsProvider>
    <div className="flex h-[100dvh] min-h-0 w-full flex-col overflow-hidden bg-white">
      <NotebookShell
        logoHref="/"
        leftByTab={leftByTab}
        workbenchTab={workbenchTab}
        onWorkbenchTabChange={handleWorkbenchTabChange}
        center={shellCenter}
        right={shellRight}
      />
      <StudioMaterialExpandedOverlay
        projectId={id}
        material={studioSidebarMaterial}
        open={studioExpandOpen}
        onMinimize={() => setStudioExpandOpen(false)}
        onCloseAll={() => {
          setStudioExpandOpen(false);
          setStudioSidebarMaterial(null);
          setChatArtifactPath(null);
        }}
      />
      <ProjectAddSourceDialog
        open={addSourceOpen}
        onOpenChange={setAddSourceOpen}
        libraryId={libraryId ?? null}
        disabled={libraryNotReady}
        onAdded={() => {
          void qc.invalidateQueries({ queryKey: ["documents", libraryId] });
        }}
      />
    </div>
    </ArtifactsProvider>
  );
}
