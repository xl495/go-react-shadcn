import type { AuthSettings, CaptchaChallenge, LoginResult, MenuNode, User } from "@/lib/types"

export const TOKEN_KEY = "latch.web.token"
export const USER_KEY = "latch.web.user"

export class ApiError extends Error {
  status: number
  code: number
  constructor(status: number, code: number, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export function isSessionExpired(err: unknown) {
  return err instanceof ApiError && (err.code === 40101 || err.code === 40102)
}

type Envelope<T> = { code: number; message: string; data?: T; errorCode?: number }

type UnauthorizedHandler = () => void
let onUnauthorized: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(fn: UnauthorizedHandler | null) {
  onUnauthorized = fn
}

function localeHeaders() {
  return localStorage.getItem("latch.web.locale") || navigator.language || "zh-CN"
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export function readStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function writeStoredUser(user: User | null) {
  if (user) localStorage.setItem(USER_KEY, JSON.stringify(user))
  else localStorage.removeItem(USER_KEY)
}

const REQUEST_TIMEOUT_MS = 12_000

function abortAfter(ms: number, parent?: AbortSignal) {
  const ctrl = new AbortController()
  const timer = window.setTimeout(() => ctrl.abort(), ms)
  const onParent = () => ctrl.abort()
  parent?.addEventListener("abort", onParent)
  return {
    signal: ctrl.signal,
    cancel() {
      window.clearTimeout(timer)
      parent?.removeEventListener("abort", onParent)
    },
  }
}

function timeoutError() {
  const zh = (localeHeaders() || "").toLowerCase().startsWith("zh")
  return new ApiError(408, 40801, zh ? "请求超时，请稍后重试" : "Request timed out. Try again.")
}

function isAbort(err: unknown) {
  return err instanceof DOMException && err.name === "AbortError"
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const locale = localeHeaders()
  headers.set("Accept", "application/json")
  headers.set("Accept-Language", locale)
  headers.set("X-Locale", locale)
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const timeout = abortAfter(REQUEST_TIMEOUT_MS, init.signal ?? undefined)
  try {
    const res = await fetch(path, { ...init, headers, signal: timeout.signal })
    const text = await res.text()
    let json: Envelope<T>
    try {
      json = text ? (JSON.parse(text) as Envelope<T>) : { code: res.ok ? 0 : 1, message: res.statusText }
    } catch {
      throw new ApiError(res.status, 1, text.trim() || res.statusText || "request failed")
    }
    if (!res.ok || json.code !== 0) {
      const err = new ApiError(res.status, json.errorCode ?? json.code ?? 1, json.message || "request failed")
      if (token && getToken() === token && isSessionExpired(err)) {
        onUnauthorized?.()
      }
      throw err
    }
    return json.data as T
  } catch (err) {
    if (isAbort(err)) throw timeoutError()
    throw err
  } finally {
    timeout.cancel()
  }
}

export const api = {
  captcha: () => request<CaptchaChallenge>("/api/v1/auth/captcha"),
  settings: () => request<AuthSettings>("/api/v1/auth/settings"),
  register: (body: {
    username: string
    email: string
    password: string
    captchaId?: string
    captchaCode?: string
    captchaToken?: string
    captchaVersion?: string
    client?: string
  }) =>
    request<LoginResult | { pending: boolean; email: string }>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ ...body, client: body.client ?? "web" }),
    }),
  verifyEmail: (body: { token: string }) =>
    request<LoginResult & { changed?: boolean }>("/api/v1/auth/verify-email", {
      method: "POST",
      body: JSON.stringify(body),
    }),
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
      body: JSON.stringify({ ...body, client: body.client ?? "web" }),
    }),
  google: (body: { idToken: string; client?: string }) =>
    request<LoginResult>("/api/v1/auth/google", {
      method: "POST",
      body: JSON.stringify({ ...body, client: body.client ?? "web" }),
    }),
  me: () => request<User>("/api/v1/auth/me"),
  webMenus: () => request<MenuNode[]>("/api/v1/auth/web-menus"),
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
      body: JSON.stringify({ ...body, client: body.client ?? "web" }),
    }),
  resetPassword: (body: { token: string; newPassword: string }) =>
    request<{ reset: boolean }>("/api/v1/auth/reset-password", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateProfile: (body: {
    nickname?: string
    email?: string
    phone?: string
    gender?: string
    department?: string
    title?: string
    remark?: string
    timezone?: string
    marketingOptIn?: boolean
  }) => request<User>("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(body) }),
  unsubscribe: (token: string) =>
    request<{ unsubscribed: boolean }>("/api/v1/mail/unsubscribe", {
      method: "POST",
      body: JSON.stringify({ token }),
    }),
  notifications: () =>
    request<{ items: NotificationItem[]; total: number }>("/api/v1/notifications?page=1&pageSize=50"),
  unreadCount: () => request<{ unread: number }>("/api/v1/notifications/unread-count"),
  readNotification: (id: number) =>
    request<NotificationItem>(`/api/v1/notifications/${id}/read`, { method: "POST" }),
  readAllNotifications: () =>
    request<{ updated: number }>("/api/v1/notifications/read-all", { method: "POST" }),
  totpSetup: (body?: { ticket?: string }) =>
    request<{ ticket: string; secret: string; otpauthUri: string }>("/api/v1/auth/totp/setup", {
      method: "POST",
      body: JSON.stringify(body ?? {}),
    }),
  totpConfirm: (body: { ticket?: string; code: string }) =>
    request<LoginResult>("/api/v1/auth/totp/confirm", { method: "POST", body: JSON.stringify(body) }),
  totpVerify: (body: { ticket: string; code?: string; recoveryCode?: string }) =>
    request<LoginResult>("/api/v1/auth/totp/verify", { method: "POST", body: JSON.stringify(body) }),
  ownSessions: () =>
    request<
      Array<{
        id: number
        ip: string
        userAgent: string
        createdAt: string
        revokedAt?: string | null
        current?: boolean
      }>
    >("/api/v1/auth/sessions"),
  revokeOwnSession: (id: number) =>
    request<{ revoked: number }>(`/api/v1/auth/sessions/${id}`, { method: "DELETE" }),
  ownLoginLogs: () =>
    request<{
      items: Array<{ id: number; status: string; ip: string; createdAt: string }>
      total: number
    }>("/api/v1/auth/login-logs?page=1&pageSize=20"),
  bindGoogle: (idToken: string) =>
    request<User>("/api/v1/auth/google/bind", { method: "POST", body: JSON.stringify({ idToken }) }),
  unbindGoogle: (body: { password?: string; totpCode?: string }) =>
    request<User>("/api/v1/auth/google/unbind", { method: "POST", body: JSON.stringify(body) }),
}

export type NotificationItem = {
  id: number
  type: string
  title: string
  body: string
  readAt?: string | null
  createdAt: string
}
