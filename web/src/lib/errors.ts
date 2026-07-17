import type { AppErrorDetail } from "@/types/api";

/**
 * 后端业务错误。对应响应信封中的 error 字段。
 * 业务错误走 HTTP 200 + success:false;鉴权/权限错误走 401/403。
 */
export class ApiError extends Error {
  readonly code: string;
  readonly httpStatus: number;
  readonly details?: AppErrorDetail[];

  constructor(
    code: string,
    message: string,
    httpStatus: number,
    details?: AppErrorDetail[],
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.httpStatus = httpStatus;
    this.details = details;
  }

  /** VALIDATION_ERROR 时按 target 字段聚合为 { field: message } */
  fieldErrors(): Record<string, string> {
    const map: Record<string, string> = {};
    for (const d of this.details ?? []) {
      if (d.target) map[d.target] = d.message;
    }
    return map;
  }

  get isAuthError(): boolean {
    return this.code === "UNAUTHORIZED" || this.code === "INVALID_TOKEN";
  }
}
