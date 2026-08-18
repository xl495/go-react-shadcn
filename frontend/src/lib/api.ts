import type {
  CaptchaChallenge,
  DashboardStats,
  DictItem,
  DictType,
  LoginResult,
  OpLog,
  Permission,
  Role,
  SysConfig,
  User,
} from "@/lib/types"

const TOKEN_KEY = "latch.token"

export class ApiError extends Error {
  status: number
  code: number
  constructor(status: number, code: number, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

type Envelope<T> = { code: number; message: string; data?: T }

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set("Accept", "application/json")
  headers.set("Accept-Language", localStorage.getItem("latch.locale") || navigator.language || "zh-CN")
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const res = await fetch(path, { ...init, headers })
  const json = (await res.json()) as Envelope<T>
  if (!res.ok || json.code !== 0) {
    throw new ApiError(res.status, json.code, json.message || "request failed")
  }
  return json.data as T
}

async function uploadAvatar(path: string, file: File): Promise<User> {
  const body = new FormData()
  body.append("file", file)
  const headers = new Headers()
  headers.set("Accept", "application/json")
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const res = await fetch(path, { method: "POST", headers, body })
  const json = (await res.json()) as Envelope<User>
  if (!res.ok || json.code !== 0) {
    throw new ApiError(res.status, json.code, json.message || "request failed")
  }
  return json.data as User
}

export const api = {
  captcha: () => request<CaptchaChallenge>("/api/v1/auth/captcha"),
  login: (body: {
    username: string
    password: string
    captchaId: string
    captchaCode: string
  }) =>
    request<LoginResult>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => request<User>("/api/v1/auth/me"),
  updateProfile: (body: {
    nickname: string
    email: string
    phone: string
    gender: string
    department: string
    title: string
    remark: string
  }) =>
    request<User>("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(body) }),
  uploadOwnAvatar: (file: File) => uploadAvatar("/api/v1/auth/avatar", file),
  uploadUserAvatar: (id: number, file: File) => uploadAvatar(`/api/v1/users/${id}/avatar`, file),
  changePassword: (body: { oldPassword: string; newPassword: string }) =>
    request<{ changed: boolean }>("/api/v1/auth/password", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  stats: () => request<DashboardStats>("/api/v1/dashboard/stats"),
  users: () => request<User[]>("/api/v1/users"),
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
    roleIds: number[]
  }) =>
    request<User>("/api/v1/users", { method: "POST", body: JSON.stringify(body) }),
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
    },
  ) => request<User>(`/api/v1/users/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteUser: (id: number) =>
    request<{ deleted: number }>(`/api/v1/users/${id}`, { method: "DELETE" }),
  assignUserRoles: (id: number, roleIds: number[]) =>
    request<User>(`/api/v1/users/${id}/roles`, {
      method: "PUT",
      body: JSON.stringify({ roleIds }),
    }),
  roles: () => request<Role[]>("/api/v1/roles"),
  createRole: (body: {
    name: string
    code: string
    description?: string
    permissionIds: number[]
  }) =>
    request<Role>("/api/v1/roles", { method: "POST", body: JSON.stringify(body) }),
  updateRole: (id: number, body: { name?: string; description?: string }) =>
    request<Role>(`/api/v1/roles/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteRole: (id: number) =>
    request<{ deleted: number }>(`/api/v1/roles/${id}`, { method: "DELETE" }),
  assignRolePermissions: (id: number, permissionIds: number[]) =>
    request<Role>(`/api/v1/roles/${id}/permissions`, {
      method: "PUT",
      body: JSON.stringify({ permissionIds }),
    }),
  permissions: () => request<Permission[]>("/api/v1/permissions"),
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
  dicts: () => request<DictType[]>("/api/v1/dicts"),
  createDict: (body: { code: string; name: string; status?: string; remark?: string }) =>
    request<DictType>("/api/v1/dicts", { method: "POST", body: JSON.stringify(body) }),
  updateDict: (id: number, body: { name?: string; status?: string; remark?: string }) =>
    request<DictType>(`/api/v1/dicts/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDict: (id: number) =>
    request<{ deleted: number }>(`/api/v1/dicts/${id}`, { method: "DELETE" }),
  dictItems: (id: number) => request<DictItem[]>(`/api/v1/dicts/${id}/items`),
  createDictItem: (
    id: number,
    body: { label: string; value: string; sort?: number; status?: string; remark?: string },
  ) => request<DictItem>(`/api/v1/dicts/${id}/items`, { method: "POST", body: JSON.stringify(body) }),
  updateDictItem: (
    id: number,
    body: { label?: string; value?: string; sort?: number; status?: string; remark?: string },
  ) => request<DictItem>(`/api/v1/dict-items/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDictItem: (id: number) =>
    request<{ deleted: number }>(`/api/v1/dict-items/${id}`, { method: "DELETE" }),
  lookupDict: (code: string) =>
    request<{ code: string; name: string; items: DictItem[] }>(`/api/v1/dicts/by/${code}`),
  configs: () => request<SysConfig[]>("/api/v1/configs"),
  createConfig: (body: { key: string; value: string; name: string; group?: string; remark?: string }) =>
    request<SysConfig>("/api/v1/configs", { method: "POST", body: JSON.stringify(body) }),
  updateConfig: (id: number, body: { value?: string; name?: string; group?: string; remark?: string }) =>
    request<SysConfig>(`/api/v1/configs/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteConfig: (id: number) =>
    request<{ deleted: number }>(`/api/v1/configs/${id}`, { method: "DELETE" }),
  logs: (params?: { username?: string; module?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.username) q.set("username", params.username)
    if (params?.module) q.set("module", params.module)
    if (params?.limit) q.set("limit", String(params.limit))
    const qs = q.toString()
    return request<OpLog[]>(`/api/v1/logs${qs ? `?${qs}` : ""}`)
  },
  clearLogs: () => request<{ cleared: boolean }>("/api/v1/logs", { method: "DELETE" }),
}
