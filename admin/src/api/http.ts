import { toast } from "sonner"
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

type Envelope<T> = { code: number; message: string; data?: T; errorCode?: number }

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

export function isSessionExpired(err: unknown) {
  return err instanceof ApiError && (err.code === 40101 || err.code === 40102)
}

function localeHeaders() {
  return localStorage.getItem("latch.locale") || navigator.language || "zh-CN"
}

function notifyApiError(err: ApiError) {
  if (isSessionExpired(err)) return
  if (err.message) toast.error(err.message)
}

async function readEnvelope<T>(res: Response): Promise<Envelope<T>> {
  const text = await res.text()
  if (!text) return { code: res.ok ? 0 : 1, message: res.statusText || "request failed" }
  try {
    return JSON.parse(text) as Envelope<T>
  } catch {
    throw new ApiError(res.status, 1, text.trim() || res.statusText || "request failed")
  }
}

function throwIfFailed<T>(res: Response, json: Envelope<T>, token: string | null, method: string): T {
  if (res.ok && json.code === 0) return json.data as T
  const err = new ApiError(res.status, json.errorCode ?? json.code ?? 1, json.message || "request failed")
  if (token && getToken() === token && isSessionExpired(err)) {
    onUnauthorized?.()
  } else if (method !== "GET" && method !== "HEAD") {
    notifyApiError(err)
  }
  throw err
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

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const locale = localeHeaders()
  headers.set("Accept", "application/json")
  headers.set("Accept-Language", locale)
  headers.set("X-Locale", locale)
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const timeout = abortAfter(REQUEST_TIMEOUT_MS, init.signal ?? undefined)
  try {
    const res = await fetch(path, { ...init, headers, signal: timeout.signal })
    const json = await readEnvelope<T>(res)
    return throwIfFailed(res, json, token, (init.method ?? "GET").toUpperCase())
  } catch (err) {
    if (isAbort(err)) throw timeoutError()
    throw err
  } finally {
    timeout.cancel()
  }
}

export async function uploadAvatar(path: string, file: File): Promise<User> {
  const body = new FormData()
  body.append("file", file)
  const headers = new Headers()
  const locale = localeHeaders()
  headers.set("Accept", "application/json")
  headers.set("Accept-Language", locale)
  headers.set("X-Locale", locale)
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  const timeout = abortAfter(REQUEST_TIMEOUT_MS)
  try {
    const res = await fetch(path, { method: "POST", headers, body, signal: timeout.signal })
    const json = await readEnvelope<User>(res)
    return throwIfFailed(res, json, token, "POST")
  } catch (err) {
    if (isAbort(err)) throw timeoutError()
    throw err
  } finally {
    timeout.cancel()
  }
}

export async function downloadCSV(path: string, filename: string) {
  const headers = new Headers()
  const token = getToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)
  headers.set("Accept-Language", localeHeaders())
  const timeout = abortAfter(REQUEST_TIMEOUT_MS)
  try {
    const res = await fetch(path, { headers, signal: timeout.signal })
    if (!res.ok) {
      let err: ApiError
      try {
        const json = await readEnvelope<unknown>(res)
        err = new ApiError(res.status, json.errorCode ?? json.code ?? 1, json.message || res.statusText)
      } catch (caught) {
        err = caught instanceof ApiError ? caught : new ApiError(res.status, 1, res.statusText)
      }
      throw err
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    if (isAbort(err)) throw timeoutError()
    throw err
  } finally {
    timeout.cancel()
  }
}

export function qs(params: Record<string, string | number | undefined>) {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ""
}
