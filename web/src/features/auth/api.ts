import { apiCall } from "@/lib/api";
import type {
  AuthTokens,
  ChangePasswordRequest,
  LoginRequest,
  User,
} from "@/types/api";

export function login(req: LoginRequest): Promise<AuthTokens> {
  return apiCall<AuthTokens>("auth/login", req, { skipAuth: true });
}

export function getMe(): Promise<User> {
  return apiCall<User>("me");
}

export function changePassword(req: ChangePasswordRequest): Promise<void> {
  return apiCall<void>("me/change-password", req);
}
