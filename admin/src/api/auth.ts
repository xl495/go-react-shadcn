import { request, uploadAvatar } from "./http"
import type { AuthSettings, CaptchaChallenge, LoginResult, MenuNode, User } from "@/types"

export const authApi = {
  captcha: () => request<CaptchaChallenge>("/api/v1/auth/captcha"),
  settings: () => request<AuthSettings>("/api/v1/auth/settings"),
  login: (body: {
    username: string
    password: string
    captchaId?: string
    captchaCode?: string
    captchaToken?: string
    captchaVersion?: string
    client?: string
  }) =>
    request<LoginResult>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  google: (body: { idToken: string; client?: string }) =>
    request<LoginResult>("/api/v1/auth/google", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => request<User>("/api/v1/auth/me"),
  menus: () => request<MenuNode[]>("/api/v1/auth/menus"),
  updateProfile: (body: {
    nickname: string
    email: string
    phone: string
    gender: string
    department: string
    title: string
    remark: string
    timezone?: string
    marketingOptIn?: boolean
  }) => request<User>("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(body) }),
  uploadOwnAvatar: (file: File) => uploadAvatar("/api/v1/auth/avatar", file),
  changePassword: (body: { oldPassword: string; newPassword: string }) =>
    request<{ changed: boolean }>("/api/v1/auth/password", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  logout: () => request<{ loggedOut: boolean }>("/api/v1/auth/logout", { method: "POST" }),
  forgotPassword: (body: {
    email: string
    client?: string
    captchaId?: string
    captchaCode?: string
    captchaToken?: string
    captchaVersion?: string
  }) =>
    request<{ sent: boolean }>("/api/v1/auth/forgot-password", {
      method: "POST",
      body: JSON.stringify({ ...body, client: body.client ?? "admin" }),
    }),
  resetPassword: (body: { token: string; newPassword: string }) =>
    request<{ reset: boolean }>("/api/v1/auth/reset-password", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  ownSessions: () => request<import("@/api/users").UserSession[]>("/api/v1/auth/sessions"),
  revokeOwnSession: (id: number) =>
    request<{ revoked: number }>(`/api/v1/auth/sessions/${id}`, { method: "DELETE" }),
  ownLoginLogs: () =>
    request<import("@/types").PageResult<import("@/types").LoginLog>>("/api/v1/auth/login-logs?page=1&pageSize=20"),
  bindGoogle: (idToken: string) =>
    request<User>("/api/v1/auth/google/bind", { method: "POST", body: JSON.stringify({ idToken }) }),
  unbindGoogle: (body: { password?: string; totpCode?: string }) =>
    request<User>("/api/v1/auth/google/unbind", { method: "POST", body: JSON.stringify(body) }),
  verifyEmail: (token: string) =>
    request<{ changed?: boolean; token?: string; user?: User }>("/api/v1/auth/verify-email", {
      method: "POST",
      body: JSON.stringify({ token }),
    }),
}
