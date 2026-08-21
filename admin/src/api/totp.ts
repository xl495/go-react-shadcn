import { request } from "./http"

export const totpApi = {
  setup: (body?: { ticket?: string }) =>
    request<{ ticket: string; secret: string; otpauthUri: string }>("/api/v1/auth/totp/setup", {
      method: "POST",
      body: JSON.stringify(body ?? {}),
    }),
  confirm: (body: { ticket?: string; code: string }) =>
    request<{ enabled?: boolean; recoveryCodes?: string[]; token?: string; user?: unknown }>(
      "/api/v1/auth/totp/confirm",
      { method: "POST", body: JSON.stringify(body) },
    ),
  verify: (body: { ticket: string; code?: string; recoveryCode?: string }) =>
    request<{ token: string; user: unknown; expiresAt: string }>("/api/v1/auth/totp/verify", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  disable: (body: { code?: string; recoveryCode?: string }) =>
    request<{ enabled: boolean }>("/api/v1/auth/totp/disable", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  recovery: (body: { code: string }) =>
    request<{ recoveryCodes: string[] }>("/api/v1/auth/totp/recovery", {
      method: "POST",
      body: JSON.stringify(body),
    }),
}
