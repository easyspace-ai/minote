import { Route, Routes } from "react-router-dom";
import { Toaster } from "sonner";

import { ProtectedRoute } from "@/auth/ProtectedRoute";
import { AppShell } from "@/layout/AppShell";
import { LandingPage } from "@/pages/LandingPage";
import { AuthPage } from "@/pages/AuthPage";
import { ProjectPage } from "@/pages/ProjectPage";
import { SkillsPage } from "@/pages/SkillsPage";
import { NewProjectRedirectPage } from "@/pages/NewProjectRedirectPage";
import { ProjectsPage } from "@/pages/ProjectsPage";
import { WorkspacePage } from "@/pages/WorkspacePage";
import { AgentChatPage } from "@/pages/workspace/agents/AgentChatPage";
import { AgentsGalleryPage } from "@/pages/workspace/agents/AgentsGalleryPage";
import { NewAgentPage } from "@/pages/workspace/agents/NewAgentPage";
import { ChatThreadPage } from "@/pages/workspace/chats/ChatThreadPage";
import { ChatsIndexPage } from "@/pages/workspace/chats/ChatsIndexPage";
import { WorkspaceHomePage } from "@/pages/workspace/WorkspaceHomePage";
import { WorkspaceLayout } from "@/pages/workspace/WorkspaceLayout";
import { SettingsPage } from "@/pages/SettingsPage";

function DocsPlaceholderPage() {
  return (
    <div className="text-muted-foreground mx-auto max-w-lg p-10 text-sm">
      文档路由已预留。完整 MDX/Nextra 迁移可在后续接入 Vite 插件或独立静态站。
    </div>
  );
}

export default function App() {
  return (
    <>
      <Routes>
        <Route path="/login" element={<AuthPage mode="login" />} />
        <Route path="/register" element={<AuthPage mode="signup" />} />
        <Route path="/landing" element={<LandingPage />} />

        <Route element={<ProtectedRoute />}>
          {/* 仅首页与文档带左侧主菜单；工作台沿用 DeerFlow 自带侧栏；项目页全屏三栏 */}
          <Route path="/" element={<AppShell />}>
            <Route index element={<WorkspacePage />} />
            <Route path="projects" element={<ProjectsPage />} />
            <Route path="new" element={<NewProjectRedirectPage />} />
            <Route path="skills" element={<SkillsPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path=":lang/docs/*" element={<DocsPlaceholderPage />} />
          </Route>

          <Route path="/workspace" element={<WorkspaceLayout />}>
            <Route index element={<WorkspaceHomePage />} />
            <Route path="chats" element={<ChatsIndexPage />} />
            <Route path="chats/:thread_id" element={<ChatThreadPage />} />
            <Route path="agents" element={<AgentsGalleryPage />} />
            <Route path="agents/new" element={<NewAgentPage />} />
            <Route
              path="agents/:agent_name/chats/:thread_id"
              element={<AgentChatPage />}
            />
          </Route>

          <Route path="/p/:projectId" element={<ProjectPage />} />
        </Route>
      </Routes>
      <Toaster position="top-center" />
    </>
  );
}
