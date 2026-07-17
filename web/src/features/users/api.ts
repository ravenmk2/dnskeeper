import { apiCall } from "@/lib/api";
import type {
  CreateUserRequest,
  UpdateUserRequest,
  User,
} from "@/types/api";

export function listUsers(): Promise<User[]> {
  return apiCall<User[]>("user/list");
}

export function createUser(req: CreateUserRequest): Promise<User> {
  return apiCall<User>("user/create", req);
}

export function updateUser(req: UpdateUserRequest): Promise<User> {
  return apiCall<User>("user/update", req);
}

export function deleteUser(id: string): Promise<void> {
  return apiCall<void>("user/delete", { id });
}
