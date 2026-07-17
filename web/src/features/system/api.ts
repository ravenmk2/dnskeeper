import { apiCall } from "@/lib/api";

/** etcd 健康检查。成功 → resolve;不可用 → 抛 SERVICE_UNAVAILABLE */
export function checkHealth(): Promise<void> {
  return apiCall<void>("health", undefined, { skipAuth: true });
}
