import { request, qs } from "./http"
import type { DashboardStats, Department, PageResult } from "@/types"

export const departmentsApi = {
  stats: () => request<DashboardStats>("/api/v1/dashboard/stats"),
  departments: (params?: { page?: number; pageSize?: number }) =>
    request<PageResult<Department>>(`/api/v1/departments${qs(params ?? {})}`),
  createDepartment: (body: {
    name: string
    code: string
    parentId?: number | null
    sort?: number
    leader?: string
    status?: string
  }) => request<Department>("/api/v1/departments", { method: "POST", body: JSON.stringify(body) }),
  updateDepartment: (
    id: number,
    body: {
      name?: string
      code?: string
      parentId?: number | null
      sort?: number
      leader?: string
      status?: string
    },
  ) => request<Department>(`/api/v1/departments/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDepartment: (id: number) =>
    request<{ deleted: number }>(`/api/v1/departments/${id}`, { method: "DELETE" }),
}
