import {
  Bell,
  ChevronRight,
  Crown,
  Database,
  Globe,
  LogOut,
  Shield,
  Sun,
  User,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { chatclawApi } from "@/api/chatclaw";
import { useAuth } from "@/auth/AuthContext";
import {
  getProfileDisplayName,
  setProfileDisplayName,
} from "@/lib/profileDisplayName";
import { cn } from "@/lib/utils";

type SettingSection = "profile" | "plan" | "preferences" | "notifications" | "data";

function ProfileSection() {
  const { user, token, setSession, refreshUser } = useAuth();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!user) return;
    setUsername(user.username?.trim() || getProfileDisplayName(user.id, user.email));
    setEmail(user.email || "");
  }, [user]);

  const handleSave = async () => {
    if (!token || !user) return;
    setSaving(true);
    try {
      const { user: u } = await chatclawApi.auth.patchMe({ email: email.trim() });
      setSession(token, u);
      setProfileDisplayName(u.id, username.trim());
      await refreshUser();
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      toast.success("已保存");
    } catch (e) {
      console.error(e);
      toast.error("保存失败，请稍后重试");
    }
    setSaving(false);
  };

  if (!user) return null;

  const displayUsername = username || getProfileDisplayName(user.id, user.email);

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">个人资料</h2>

      <div className="flex items-center gap-6 rounded-xl border border-gray-200 bg-white p-6">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gray-100">
          <span className="text-2xl font-semibold text-gray-500">
            {(displayUsername || "U").charAt(0).toUpperCase()}
          </span>
        </div>
        <div>
          <p className="text-sm font-medium text-gray-900">{displayUsername || "用户"}</p>
          <p className="mt-0.5 text-xs text-gray-400">{email || ""}</p>
        </div>
      </div>

      <div className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-6">
        <div>
          <h3 className="text-sm font-medium text-gray-900">当前可用积分</h3>
          <p className="mt-1 text-2xl font-bold text-primary">
            {user.credits_balance != null ? Math.floor(user.credits_balance) : "—"}
          </p>
          <p className="mt-1 text-xs text-gray-400">
            已消耗: {user.credits_used != null ? Math.floor(user.credits_used) : 0} 积分
          </p>
        </div>
        <button
          type="button"
          className="rounded-lg bg-primary/10 px-4 py-2 text-sm font-medium text-primary transition-colors hover:bg-primary/15"
        >
          充值积分
        </button>
      </div>

      <div className="space-y-4 rounded-xl border border-gray-200 bg-white p-6">
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700">用户名</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm transition-all focus:border-gray-400 focus:ring-1 focus:ring-gray-200 focus:outline-none"
          />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700">邮箱</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm transition-all focus:border-gray-400 focus:ring-1 focus:ring-gray-200 focus:outline-none"
          />
        </div>
        <div className="pt-2">
          <button
            type="button"
            onClick={handleSave}
            disabled={saving}
            className="rounded-lg bg-gray-900 px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:opacity-50"
          >
            {saving ? "保存中..." : saved ? "✓ 已保存" : "保存更改"}
          </button>
        </div>
      </div>
    </div>
  );
}

export function SettingsPage() {
  const navigate = useNavigate();
  const { logout } = useAuth();
  const [activeSection, setActiveSection] = useState<SettingSection>("profile");
  const [notifications, setNotifications] = useState({
    email: true,
    push: false,
    weekly: true,
  });

  const onLogout = useCallback(() => {
    logout();
    navigate("/login", { replace: true });
  }, [logout, navigate]);

  const sections = [
    { id: "profile" as const, icon: User, label: "个人资料" },
    { id: "plan" as const, icon: Crown, label: "套餐管理" },
    { id: "preferences" as const, icon: Shield, label: "偏好设置" },
    { id: "notifications" as const, icon: Bell, label: "通知设置" },
    { id: "data" as const, icon: Database, label: "数据管理" },
  ];

  return (
    <div className="flex h-full min-h-0 bg-white">
      <div className="flex w-64 shrink-0 flex-col border-r border-gray-200 bg-white p-4">
        <h1 className="mb-6 px-4 text-xl font-semibold">设置</h1>
        <nav className="min-h-0 flex-1 space-y-1 overflow-y-auto">
          {sections.map((section) => (
            <button
              key={section.id}
              type="button"
              onClick={() => setActiveSection(section.id)}
              className={cn(
                "flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left transition-colors",
                activeSection === section.id
                  ? "bg-gray-100 font-medium"
                  : "text-gray-600 hover:bg-gray-50",
              )}
            >
              <section.icon size={18} />
              <span>{section.label}</span>
            </button>
          ))}
        </nav>

        <div className="shrink-0 border-t border-gray-100 pt-4">
          <button
            type="button"
            onClick={onLogout}
            className="flex w-full items-center gap-2 rounded-lg px-4 py-2 text-left text-gray-500 hover:bg-gray-50 hover:text-gray-700"
          >
            <LogOut size={18} />
            <span>退出登录</span>
          </button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-8">
        <div className="max-w-2xl">
          {activeSection === "profile" && <ProfileSection />}

          {activeSection === "plan" && (
            <div className="space-y-6">
              <h2 className="text-lg font-semibold">套餐管理</h2>

              <div className="border border-gray-200 bg-white p-6">
                <div className="mb-4 flex items-center justify-between">
                  <div>
                    <div className="text-sm text-gray-500">当前套餐</div>
                    <div className="text-xl font-semibold">免费版</div>
                  </div>
                  <button
                    type="button"
                    className="bg-gray-900 px-4 py-2 text-white hover:bg-gray-800"
                  >
                    升级套餐
                  </button>
                </div>
                <div className="grid grid-cols-3 gap-4 border-t border-gray-200 pt-4">
                  <div>
                    <div className="text-2xl font-semibold">100</div>
                    <div className="text-sm text-gray-500">剩余积分</div>
                  </div>
                  <div>
                    <div className="text-2xl font-semibold">3</div>
                    <div className="text-sm text-gray-500">项目数</div>
                  </div>
                  <div>
                    <div className="text-2xl font-semibold">5</div>
                    <div className="text-sm text-gray-500">已安装技能</div>
                  </div>
                </div>
              </div>

              <div className="border border-gray-200 bg-white p-6">
                <h3 className="mb-4 font-medium">套餐对比</h3>
                <div className="grid grid-cols-3 gap-4">
                  <div className="border border-gray-200 p-4">
                    <div className="font-medium">免费版</div>
                    <div className="my-2 text-2xl font-semibold">¥0</div>
                    <div className="text-xs text-gray-500">永久免费</div>
                    <ul className="mt-4 space-y-2 text-sm text-gray-600">
                      <li>✓ 100 积分/月</li>
                      <li>✓ 5 个项目</li>
                      <li>✓ 基础技能</li>
                    </ul>
                  </div>
                  <div className="relative border-2 border-gray-900 p-4">
                    <div className="absolute -top-3 left-4 bg-gray-900 px-2 text-xs text-white">
                      推荐
                    </div>
                    <div className="font-medium">专业版</div>
                    <div className="my-2 text-2xl font-semibold">¥49</div>
                    <div className="text-xs text-gray-500">每月</div>
                    <ul className="mt-4 space-y-2 text-sm text-gray-600">
                      <li>✓ 1000 积分/月</li>
                      <li>✓ 无限项目</li>
                      <li>✓ 所有技能</li>
                    </ul>
                  </div>
                  <div className="border border-gray-200 p-4">
                    <div className="font-medium">团队版</div>
                    <div className="my-2 text-2xl font-semibold">¥199</div>
                    <div className="text-xs text-gray-500">每月</div>
                    <ul className="mt-4 space-y-2 text-sm text-gray-600">
                      <li>✓ 无限积分</li>
                      <li>✓ 团队协作</li>
                      <li>✓ 优先支持</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeSection === "preferences" && (
            <div className="space-y-6">
              <h2 className="text-lg font-semibold">偏好设置</h2>

              <div className="divide-y divide-gray-200 border border-gray-200 bg-white">
                <div className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-3">
                    <Sun size={20} />
                    <div>
                      <p className="font-medium">主题模式</p>
                      <p className="text-sm text-gray-500">选择浅色或深色主题</p>
                    </div>
                  </div>
                  <button
                    type="button"
                    className={cn(
                      "relative h-8 w-14 rounded-full transition-colors",
                      false ? "bg-gray-900" : "bg-gray-300",
                    )}
                  >
                    <span
                      className={cn(
                        "absolute top-1 h-6 w-6 rounded-full bg-white transition-transform",
                        false ? "translate-x-7" : "translate-x-1",
                      )}
                    />
                  </button>
                </div>

                <div className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-3">
                    <Globe size={20} />
                    <div>
                      <p className="font-medium">语言</p>
                      <p className="text-sm text-gray-500">界面显示语言</p>
                    </div>
                  </div>
                  <select className="border border-gray-300 bg-white px-4 py-2">
                    <option>简体中文</option>
                    <option>English</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {activeSection === "notifications" && (
            <div className="space-y-6">
              <h2 className="text-lg font-semibold">通知设置</h2>

              <div className="divide-y divide-gray-200 border border-gray-200 bg-white">
                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">邮件通知</p>
                    <p className="text-sm text-gray-500">接收重要更新邮件</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setNotifications({ ...notifications, email: !notifications.email })}
                    className={cn(
                      "relative h-8 w-14 rounded-full transition-colors",
                      notifications.email ? "bg-gray-900" : "bg-gray-300",
                    )}
                  >
                    <span
                      className={cn(
                        "absolute top-1 h-6 w-6 rounded-full bg-white transition-transform",
                        notifications.email ? "translate-x-7" : "translate-x-1",
                      )}
                    />
                  </button>
                </div>

                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">推送通知</p>
                    <p className="text-sm text-gray-500">浏览器推送通知</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setNotifications({ ...notifications, push: !notifications.push })}
                    className={cn(
                      "relative h-8 w-14 rounded-full transition-colors",
                      notifications.push ? "bg-gray-900" : "bg-gray-300",
                    )}
                  >
                    <span
                      className={cn(
                        "absolute top-1 h-6 w-6 rounded-full bg-white transition-transform",
                        notifications.push ? "translate-x-7" : "translate-x-1",
                      )}
                    />
                  </button>
                </div>

                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">每周摘要</p>
                    <p className="text-sm text-gray-500">每周项目活动摘要</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setNotifications({ ...notifications, weekly: !notifications.weekly })}
                    className={cn(
                      "relative h-8 w-14 rounded-full transition-colors",
                      notifications.weekly ? "bg-gray-900" : "bg-gray-300",
                    )}
                  >
                    <span
                      className={cn(
                        "absolute top-1 h-6 w-6 rounded-full bg-white transition-transform",
                        notifications.weekly ? "translate-x-7" : "translate-x-1",
                      )}
                    />
                  </button>
                </div>
              </div>
            </div>
          )}

          {activeSection === "data" && (
            <div className="space-y-6">
              <h2 className="text-lg font-semibold">数据管理</h2>

              <div className="divide-y divide-gray-200 border border-gray-200 bg-white">
                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">导出数据</p>
                    <p className="text-sm text-gray-500">导出所有项目数据</p>
                  </div>
                  <button
                    type="button"
                    className="flex items-center gap-2 border border-gray-300 px-4 py-2 hover:bg-gray-50"
                  >
                    导出
                    <ChevronRight size={16} />
                  </button>
                </div>

                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium">存储空间</p>
                    <p className="text-sm text-gray-500">已使用 128 MB / 1 GB</p>
                  </div>
                  <div className="h-2 w-32 overflow-hidden rounded-full bg-gray-200">
                    <div className="h-full w-1/4 bg-gray-900" />
                  </div>
                </div>

                <div className="flex items-center justify-between p-4">
                  <div>
                    <p className="font-medium text-red-500">删除账号</p>
                    <p className="text-sm text-gray-500">永久删除账号和所有数据</p>
                  </div>
                  <button
                    type="button"
                    className="border border-red-300 px-4 py-2 text-red-500 hover:bg-red-50"
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
