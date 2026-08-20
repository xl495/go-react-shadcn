import { request, qs } from "./http"
import type { PageResult, Role } from "@/types"

export const rolesApi = {
  roles: (params?: { page?: number; pageSize?: number }) =>
    request<PageResult<Role>>(`/api/v1/roles${qs(params ?? {})}`),
  createRole: (body: {
    name: string
    code: string
    description?: string
    dataScope?: string
    permissionIds: number[]
  }) => request<Role>("/api/v1/roles", { method: "POST", body: JSON.stringify(body) }),
  updateRole: (id: number, body: { name?: string; description?: string; dataScope?: string }) =>
    request<Role>(`/api/v1/roles/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteRole: (id: number) => request<{ deleted: number }>(`/api/v1/roles/${id}`, { method: "DELETE" }),
  assignRolePermissions: (id: number, permissionIds: number[]) =>
    request<Role>(`/api/v1/roles/${id}/permissions`, {
      method: "PUT",
      body: JSON.stringify({ permissionIds }),
    }),
}
