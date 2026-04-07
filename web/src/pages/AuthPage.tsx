/**
 * Layout inspired by okui AuthPage: split brand panel + form (Chinese copy).
 */

import { useEffect, useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { motion } from "motion/react";
import { Sparkles, Mail, Lock, LogIn, UserPlus } from "lucide-react";
import { chatclawApi } from "@/api/chatclaw";
import { ApiError } from "@/api/http";
import { useAuth } from "@/auth/AuthContext";

type AuthMode = "login" | "signup";

export function AuthPage({ mode: initialMode }: { mode: AuthMode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const { setSession, ready, token } = useAuth();
  const [mode, setMode] = useState<AuthMode>(initialMode);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    setMode(initialMode);
  }, [initialMode]);

  if (ready && token) {
    const from = (location.state as { from?: string } | null)?.from;
    return <Navigate to={from && from !== "/login" ? from : "/"} replace />;
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setPending(true);
    try {
      const res =
        mode === "login"
          ? await chatclawApi.auth.login({ email, password })
          : await chatclawApi.auth.register({ email, password });
      setSession(res.token, res.user);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from && from !== "/login" ? from : "/", { replace: true });
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : "请求失败";
      setError(msg);
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex min-h-screen w-full overflow-hidden bg-white">
      <div className="relative hidden w-1/2 items-center justify-center overflow-hidden bg-black p-20 lg:flex">
        <div className="absolute inset-0 opacity-40">
          <img
            src="https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&q=80&w=1200"
            alt=""
            className="h-full w-full object-cover"
            referrerPolicy="no-referrer"
          />
        </div>
        <div className="relative z-10 max-w-lg">
          <div className="mb-10 flex h-16 w-16 items-center justify-center rounded-2xl bg-white shadow-2xl">
            <Sparkles className="h-8 w-8 fill-black text-black" />
          </div>
        
          <p className="text-lg font-medium leading-relaxed text-white/60">
            面向研究、合成与创作的本地智能工作区。
          </p>
        </div>
      </div>

      <div className="relative flex w-full flex-col items-center justify-center p-8 md:p-20 lg:w-1/2">
        <div className="absolute right-10 top-10 lg:hidden">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-black">
            <Sparkles className="h-5 w-5 fill-white text-white" />
          </div>
        </div>

        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          className="w-full max-w-md"
        >
          <div className="mb-10">
            <h2 className="mb-3 text-4xl font-black tracking-tight text-black">
              {mode === "login" ? "欢迎回来" : "创建账号"}
            </h2>
            <p className="text-lg text-muted-foreground">
              {mode === "login" ? "登录以继续使用工作区。" : "注册后开始使用本地控制台。"}
            </p>
          </div>

          {error ? (
            <p className="mb-4 rounded-xl border border-outline-variant/30 bg-surface-container-low px-4 py-3 text-sm text-on-surface">
              {error}
            </p>
          ) : null}

          <form onSubmit={onSubmit} className="space-y-6">
            <div className="space-y-2">
              <label className="ml-1 text-xs font-bold uppercase tracking-widest text-muted-foreground">
                邮箱
              </label>
              <div className="group relative">
                <Mail className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground transition-colors group-focus-within:text-black" />
                <input
                  type="email"
                  required
                  autoComplete="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full rounded-2xl border-none bg-surface-container-low py-4 pl-12 pr-4 text-black placeholder:text-muted-foreground/30 focus:ring-2 focus:ring-black/5"
                  placeholder="name@example.com"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="ml-1 text-xs font-bold uppercase tracking-widest text-muted-foreground">
                密码
              </label>
              <div className="group relative">
                <Lock className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-muted-foreground transition-colors group-focus-within:text-black" />
                <input
                  type="password"
                  required
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-2xl border-none bg-surface-container-low py-4 pl-12 pr-4 text-black placeholder:text-muted-foreground/30 focus:ring-2 focus:ring-black/5"
                  placeholder="至少 8 位字符"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={pending}
              className="flex w-full items-center justify-center gap-3 rounded-2xl bg-black py-4 text-lg font-bold text-white shadow-xl transition-all hover:opacity-90 active:scale-[0.98] disabled:opacity-60"
            >
              {mode === "login" ? <LogIn className="h-5 w-5" /> : <UserPlus className="h-5 w-5" />}
              {pending ? "请稍候…" : mode === "login" ? "登录" : "注册"}
            </button>
          </form>

          <div className="mt-10">
            <div className="relative flex items-center py-5">
              <div className="flex-grow border-t border-outline-variant/10" />
              <span className="mx-4 flex-shrink text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                第三方登录（即将推出）
              </span>
              <div className="flex-grow border-t border-outline-variant/10" />
            </div>
            <div className="grid grid-cols-2 gap-4 opacity-50">
              <button
                type="button"
                disabled
                className="flex items-center justify-center gap-3 rounded-xl border border-outline-variant/10 py-3 px-4 text-sm font-bold"
              >
                Google
              </button>
              <button
                type="button"
                disabled
                className="flex items-center justify-center gap-3 rounded-xl border border-outline-variant/10 py-3 px-4 text-sm font-bold"
              >
                GitHub
              </button>
            </div>
          </div>

          <p className="mt-12 text-center text-sm text-muted-foreground">
            {mode === "login" ? "还没有账号？" : "已有账号？"}{" "}
            {mode === "login" ? (
              <Link to="/register" className="font-bold text-black hover:underline">
                免费注册
              </Link>
            ) : (
              <Link to="/login" className="font-bold text-black hover:underline">
                去登录
              </Link>
            )}
          </p>
        </motion.div>
      </div>
    </div>
  );
}
