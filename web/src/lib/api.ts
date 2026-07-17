import { useAuthStore } from "@/stores/auth";
import type { AppError, Envelope } from "@/types/api";

import { ApiError } from "./errors";

const BASE = "/api";

/** 刷新令牌的单飞 Promise:并发 401 共享同一次刷新 */
let refreshPromise: Promise<string> | null = null;

function redirectLogin() {
  if (!window.location.pathname.startsWith("/login")) {
    window.location.assign("/login");
  }
}

async function refreshTokens(): Promise<string> {
  if (refreshPromise) return refreshPromise;
  const rt = useAuthStore.getState().refreshToken;
  if (!rt) {
    throw new ApiError("INVALID_TOKEN", "登录已过期,请重新登录", 401);
  }
  refreshPromise = (async () => {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    const env = (await res
      .json()
      .catch(() => null)) as Envelope<{
      token: string;
      refresh_token: string;
    }> | null;
    if (!res.ok || !env || !env.success || !env.data) {
      useAuthStore.getState().logout();
      redirectLogin();
      throw new ApiError(
        env?.error?.code ?? "INVALID_TOKEN",
        env?.error?.message ?? "登录已过期,请重新登录",
        res.status,
      );
    }
    useAuthStore.getState().setTokens(env.data.token, env.data.refresh_token);
    return env.data.token;
  })().finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

export interface ApiCallOptions {
  /** 跳过注入 Bearer 与 401 自动刷新(用于 login / refresh 自身) */
  skipAuth?: boolean;
}

/**
 * 调用后端 RPC。统一 POST + JSON body,自动解包信封。
 * - HTTP 200 + success:true → 返回 data
 * - HTTP 200 + success:false → 抛 ApiError(业务错误)
 * - HTTP 401 → 单飞刷新一次后重试;刷新失败 → logout 并跳 /login
 * - HTTP 403/500/503 → 抛 ApiError
 */
export async function apiCall<T>(
  path: string,
  body?: unknown,
  opts: ApiCallOptions = {},
): Promise<T> {
  const token = opts.skipAuth ? null : useAuthStore.getState().token;

  const send = (tok: string | null): Promise<Response> =>
    fetch(`${BASE}/${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(tok ? { Authorization: `Bearer ${tok}` } : {}),
      },
      body: body === undefined ? "{}" : JSON.stringify(body),
    });

  let res = await send(token);

  if (res.status === 401 && !opts.skipAuth && token) {
    try {
      const newToken = await refreshTokens();
      res = await send(newToken);
    } catch {
      throw new ApiError("UNAUTHORIZED", "登录已过期,请重新登录", 401);
    }
  }

  const env = (await res
    .json()
    .catch(() => null)) as Envelope<T> | null;
  if (!env) {
    throw new ApiError("INTERNAL_ERROR", "响应解析失败", res.status);
  }
  if (!env.success || env.error) {
    const e: AppError | null = env.error;
    throw new ApiError(
      e?.code ?? "INTERNAL_ERROR",
      e?.message ?? "未知错误",
      res.status,
      e?.details,
    );
  }
  return env.data as T;
}
