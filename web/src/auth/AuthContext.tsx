import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";
import { chatclawApi, type AuthUser, type MetaResponse } from "@/api/chatclaw";
import { setAuthTokenGetter, setOnUnauthorized } from "@/api/http";
import { readStoredToken, writeStoredToken } from "./authStorage";

type AuthState = {
  token: string | null;
  user: AuthUser | null;
  meta: MetaResponse | null;
  ready: boolean;
  /** Server or client requires login before using the app. */
  needLogin: boolean;
  setSession: (token: string | null, user: AuthUser | null) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

/** Optional env override to force login in the web UI even if meta says otherwise. */
function clientRequiresAuth(): boolean {
  return import.meta.env.VITE_REQUIRE_AUTH === "true";
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const [token, setToken] = useState<string | null>(() => readStoredToken());
  const [user, setUser] = useState<AuthUser | null>(null);
  const [meta, setMeta] = useState<MetaResponse | null>(null);
  const [ready, setReady] = useState(false);
  const tokenRef = useRef<string | null>(readStoredToken());

  useEffect(() => {
    tokenRef.current = token;
  }, [token]);

  const setSession = useCallback((t: string | null, u: AuthUser | null) => {
    setToken(t);
    setUser(u);
    writeStoredToken(t);
  }, []);

  const logout = useCallback(() => {
    setSession(null, null);
  }, [setSession]);

  const refreshUser = useCallback(async () => {
    const t = tokenRef.current;
    if (!t) {
      setUser(null);
      return;
    }
    try {
      const { user: u } = await chatclawApi.auth.me();
      setUser(u);
    } catch {
      setSession(null, null);
    }
  }, [setSession]);

  useEffect(() => {
    setAuthTokenGetter(() => tokenRef.current);
  }, []);

  useEffect(() => {
    if (!ready) {
      setOnUnauthorized(null);
      return;
    }
    setOnUnauthorized(() => {
      writeStoredToken(null);
      setToken(null);
      setUser(null);
      navigate("/login", { replace: true });
    });
    return () => setOnUnauthorized(null);
  }, [ready, navigate]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const m = await chatclawApi.meta();
        if (cancelled) return;
        setMeta(m);
        const t = readStoredToken();
        if (t) {
          tokenRef.current = t;
          setToken(t);
          try {
            const { user: u } = await chatclawApi.auth.me();
            if (!cancelled) setUser(u);
          } catch {
            if (!cancelled) setSession(null, null);
          }
        }
      } catch {
        if (!cancelled) setMeta(null);
      } finally {
        if (!cancelled) setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [setSession]);

  // Server defaults to auth on; treat unknown meta as requiring login.
  const needLogin = clientRequiresAuth() || meta?.auth_required !== false;

  const value = useMemo<AuthState>(
    () => ({
      token,
      user,
      meta,
      ready,
      needLogin,
      setSession,
      logout,
      refreshUser,
    }),
    [token, user, meta, ready, needLogin, setSession, logout, refreshUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
