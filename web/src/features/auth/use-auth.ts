import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { ApiError } from "@/lib/errors";
import { useAuthStore } from "@/stores/auth";
import type { ChangePasswordRequest, LoginRequest } from "@/types/api";

import { changePassword, getMe, login } from "./api";

/**
 * 登录流程:login → setTokens → getMe → setUser。
 * 页面层在 mutate 的 onSuccess 里做跳转、onError 里做错误展示。
 */
export function useLogin() {
  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);

  return useMutation({
    mutationFn: async (req: LoginRequest) => {
      const tokens = await login(req);
      // 先落 token,使后续 getMe 能带上 Bearer
      setTokens(tokens.token, tokens.refresh_token);
      const me = await getMe();
      setUser(me);
      return me;
    },
  });
}

/** 引导校验:有 token 但无 user 时拉取 /api/me(覆盖刷新页面后 user 丢失等边角) */
export function useMe() {
  const token = useAuthStore((s) => s.token);
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);

  return useQuery({
    queryKey: ["me"],
    queryFn: async () => {
      const me = await getMe();
      setUser(me);
      return me;
    },
    enabled: !!token && !user,
  });
}

/** 修改当前用户密码。错误(旧密码错/相同/弱)由调用方处理 */
export function useChangePassword() {
  return useMutation({
    mutationFn: (req: ChangePasswordRequest) => changePassword(req),
    onError: (err: unknown) => {
      const e = err as ApiError;
      if (
        e.code !== "WRONG_PASSWORD" &&
        e.code !== "SAME_PASSWORD" &&
        e.code !== "WEAK_PASSWORD"
      ) {
        toast.error(e.message || "修改失败");
      }
    },
  });
}
