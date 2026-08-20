import type { User } from "@/types"

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

type UnauthorizedHandler = () => void
let onUnauthorized: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(fn: UnauthorizedHandler | null) {
  onUnauthorized = fn
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
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
    const err = new ApiError(res.status, json.code, json.message || "request failed")
    if (err.status === 401 || err.code === 40101 || err.code === 40102) {
      onUnauthorized?.()
    }
    throw err
  }
  return json.data as T
}

export async function uploadAvatar(path: string, file: File): Promise<User> {
  const body = new FormData()
  body.append("file", file)
  const headers = new Headers()
  headers.set("Accept", "application/json")
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const res = await fetch(path, { method: "POST", headers, body })
  const json = (await res.json()) as Envelope<User>
  if (!res.ok || json.code !== 0) {
    const err = new ApiError(res.status, json.code, json.message || "request failed")
    if (err.status === 401 || err.code === 40101 || err.code === 40102) onUnauthorized?.()
    throw err
  }
  return json.data as User
}

export function qs(params: Record<string, string | number | undefined>) {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ""
}
