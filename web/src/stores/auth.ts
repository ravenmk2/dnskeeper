import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { User } from "@/types/api";

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  /** 由登录/刷新流程填充 */
  setTokens: (token: string, refreshToken: string) => void;
  setUser: (user: User | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      setTokens: (token, refreshToken) => set({ token, refreshToken }),
      setUser: (user) => set({ user }),
      logout: () => set({ token: null, refreshToken: null, user: null }),
    }),
    {
      name: "dnskeeper-auth",
      // 仅持久化 token / refreshToken / user
      partialize: (s) => ({
        token: s.token,
        refreshToken: s.refreshToken,
        user: s.user,
      }),
    },
  ),
);

export function isAuthenticated(): boolean {
  return useAuthStore.getState().token !== null;
}

export function isAdmin(): boolean {
  return useAuthStore.getState().user?.user_type === "admin";
}
