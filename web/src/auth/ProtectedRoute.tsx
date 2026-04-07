import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "./AuthContext";

export function ProtectedRoute() {
  const location = useLocation();
  const { ready, needLogin, token } = useAuth();

  if (!ready) {
    return (
      <div className="flex h-[100dvh] items-center justify-center bg-surface text-sm text-muted-foreground">
        加载中…
      </div>
    );
  }

  if (needLogin && !token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}
