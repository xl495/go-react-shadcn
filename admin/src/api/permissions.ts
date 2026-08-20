import { request, qs } from "./http"
import type { PageResult, Permission } from "@/types"

export const permissionsApi = {
  permissions: (params?: { page?: number; pageSize?: number; q?: string; kind?: string }) =>
    request<PageResult<Permission>>(`/api/v1/permissions${qs(params ?? {})}`),
  createPermission: (body: {
    name: string
    code: string
    path: string
    method: string
    kind?: string
    description?: string
  }) =>
    request<Permission>("/api/v1/permissions", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updatePermission: (
    id: number,
    body: { name?: string; path?: string; method?: string; description?: string },
  ) =>
    request<Permission>(`/api/v1/permissions/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deletePermission: (id: number) =>
    request<{ deleted: number }>(`/api/v1/permissions/${id}`, { method: "DELETE" }),
}
