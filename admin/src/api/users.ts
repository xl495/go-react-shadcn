import { request, uploadAvatar, qs } from "./http"
import type { PageResult, User } from "@/types"

export const usersApi = {
  users: (params?: {
    page?: number
    pageSize?: number
    q?: string
    status?: string
    gender?: string
    department?: string
    roleId?: number
    kind?: string
  }) => request<PageResult<User>>(`/api/v1/users${qs(params ?? {})}`),
  getUser: (id: number) => request<User>(`/api/v1/users/${id}`),
  createUser: (body: {
    username: string
    password: string
    status?: string
    nickname?: string
    email?: string
    phone?: string
    gender?: string
    department?: string
    title?: string
    remark?: string
    timezone?: string
    marketingOptIn?: boolean
    kind?: string
    roleIds: number[]
  }) => request<User>("/api/v1/users", { method: "POST", body: JSON.stringify(body) }),
  updateUser: (
    id: number,
    body: {
      password?: string
      status?: string
      nickname?: string
      email?: string
      phone?: string
      gender?: string
      department?: string
      title?: string
      remark?: string
      timezone?: string
      marketingOptIn?: boolean
    },
  ) => request<User>(`/api/v1/users/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteUser: (id: number) => request<{ deleted: number }>(`/api/v1/users/${id}`, { method: "DELETE" }),
  assignUserRoles: (id: number, roleIds: number[]) =>
    request<User>(`/api/v1/users/${id}/roles`, {
      method: "PUT",
      body: JSON.stringify({ roleIds }),
    }),
  uploadUserAvatar: (id: number, file: File) => uploadAvatar(`/api/v1/users/${id}/avatar`, file),
}
