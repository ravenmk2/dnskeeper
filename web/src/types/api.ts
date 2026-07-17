// 后端 API 数据模型 — 与 internal/store/store.go 对齐
// 响应中 password / last_record_id 不暴露,故此处省略

export type UserType = "admin" | "normal";

export interface User {
  id: string;
  username: string;
  user_type: UserType;
  builtin: boolean;
  created_at: string;
  updated_at: string;
}

export interface Zone {
  zone: string;
  domain_count: number;
  created_at: string;
  updated_at: string;
}

export interface Domain {
  zone: string;
  domain: string;
  name: string;
  record_count: number;
  created_at: string;
  updated_at: string;
}

export type RecordType = "A" | "AAAA" | "SRV" | "TXT";

export interface Record {
  id: string;
  type: RecordType;
  value: string;
  ttl: number;
  /** 仅 SRV */
  priority?: number;
  /** 仅 SRV */
  port?: number;
  /** 仅 SRV,缺省 0 */
  weight?: number;
}

// 统一响应信封 { success, data, error }
export interface AppErrorDetail {
  code: string;
  message: string;
  target: string;
}

export interface AppError {
  code: string;
  message: string;
  details?: AppErrorDetail[];
}

export interface Envelope<T> {
  success: boolean;
  data: T | null;
  error: AppError | null;
}

// 认证响应 data
export interface AuthTokens {
  token: string;
  refresh_token: string;
}

// 请求体类型
export interface LoginRequest {
  username: string;
  password: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  user_type: UserType;
}

export interface UpdateUserRequest {
  id: string;
  password?: string;
  user_type?: UserType;
}

export interface CreateZoneRequest {
  zone: string;
}

export interface CreateDomainRequest {
  zone: string;
  domain: string;
}

export interface CreateRecordRequest {
  zone: string;
  domain: string;
  type: RecordType;
  value: string;
  ttl: number;
  priority?: number;
  port?: number;
  weight?: number;
}

export interface UpdateRecordRequest {
  zone: string;
  domain: string;
  id: string;
  value: string;
  ttl: number;
  priority?: number;
  port?: number;
  weight?: number;
}
