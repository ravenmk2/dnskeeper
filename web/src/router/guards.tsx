import { Navigate, useLocation } from "react-router-dom";

import { useAuthStore } from "@/stores/auth";

/** 未认证 → 跳登录(带来源 state) */
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  const location = useLocation();
  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return <>{children}</>;
}

/** 非 admin → 跳 403 */
export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const userType = useAuthStore((s) => s.user?.user_type);
  if (userType !== "admin") {
    return <Navigate to="/403" replace />;
  }
  return <>{children}</>;
}
