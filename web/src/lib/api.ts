import type { CaptchaChallenge, LoginResult, User } from "@/lib/types"

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

type Envelope<T> = { code: number; message: string; data?: T }

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

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set("Accept", "application/json")
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const res = await fetch(path, { ...init, headers })
  let json: Envelope<T>
  try {
    json = (await res.json()) as Envelope<T>
  } catch {
    throw new ApiError(res.status, -1, "invalid response")
  }
  if (!res.ok || json.code !== 0) {
    throw new ApiError(res.status, json.code, json.message || "request failed")
  }
  return json.data as T
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
  changePassword: (body: { oldPassword: string; newPassword: string }) =>
    request<{ changed: boolean }>("/api/v1/auth/password", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  forgotPassword: (body: { email: string; captchaId: string; captchaCode: string }) =>
    request<{ sent: boolean }>("/api/v1/auth/forgot-password", {
      method: "POST",
      body: JSON.stringify(body),
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
}
