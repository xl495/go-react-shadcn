import { request, uploadAvatar, qs, downloadCSV } from "./http"
import type { PageResult, User } from "@/types"

export type UserSession = {
  id: number
  userId: number
  userKind: string
  ip: string
  userAgent: string
  expiresAt: string
  revokedAt?: string | null
  createdAt: string
}

export type UserImportJob = {
  id: number
  actorId: number
  kind: string
  fileName: string
  status: string
  total: number
  created: number
  failed: number
  errors?: string
  createdAt: string
  updatedAt: string
}

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
  getUser: (id: number, kind?: string) => request<User>(`/api/v1/users/${id}${qs({ kind })}`),
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
      kind?: string
    },
  ) => request<User>(`/api/v1/users/${id}${qs({ kind: body.kind })}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteUser: (id: number, kind?: string) =>
    request<{ deleted: number }>(`/api/v1/users/${id}${qs({ kind })}`, { method: "DELETE" }),
  assignUserRoles: (id: number, roleIds: number[], kind?: string) =>
    request<User>(`/api/v1/users/${id}/roles${qs({ kind })}`, {
      method: "PUT",
      body: JSON.stringify({ roleIds }),
    }),
  uploadUserAvatar: (id: number, file: File, kind?: string) =>
    uploadAvatar(`/api/v1/users/${id}/avatar${qs({ kind })}`, file),
  revokeUser: (id: number, kind?: string) =>
    request<{ revoked: number }>(`/api/v1/users/${id}/revoke${qs({ kind })}`, { method: "POST" }),
  userSessions: (id: number, kind?: string) =>
    request<UserSession[]>(`/api/v1/users/${id}/sessions${qs({ kind })}`),
  revokeUserSession: (id: number, sid: number, kind?: string) =>
    request<{ revoked: number }>(`/api/v1/users/${id}/sessions/${sid}${qs({ kind })}`, { method: "DELETE" }),
  exportUsers: (kind?: string) => downloadCSV(`/api/v1/users/export${qs({ kind })}`, "users.csv"),
  importUsers: (file: File, kind?: string) => {
    const body = new FormData()
    body.append("file", file)
    return request<UserImportJob>(`/api/v1/users/import${qs({ kind })}`, { method: "POST", body })
  },
  importUserJob: (id: number) => request<UserImportJob>(`/api/v1/users/import-jobs/${id}`),
  resetUserPassword: (id: number, kind?: string) =>
    request<{ temporaryPassword: string; mustChangePassword: boolean }>(
      `/api/v1/users/${id}/reset-password${qs({ kind })}`,
      { method: "POST" },
    ),
  unlockUser: (id: number, kind?: string) =>
    request<{ unlocked: number }>(`/api/v1/users/${id}/unlock${qs({ kind })}`, { method: "POST" }),
  batchUserStatus: (body: { ids: number[]; status: string; kind?: string }) =>
    request<{ updated: number[] }>("/api/v1/users/batch-status", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  onlineSessions: () =>
    request<
      Array<{
        id: number
        userId: number
        userKind: string
        username: string
        ip: string
        userAgent: string
        expiresAt: string
        createdAt: string
      }>
    >("/api/v1/online-sessions"),
}
