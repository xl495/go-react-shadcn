import { request, qs, downloadCSV } from "./http"
import type { APILog, LoginLog, OpLog, PageResult } from "@/types"

export const logsApi = {
  logs: (params?: { username?: string; module?: string; action?: string; page?: number; pageSize?: number }) =>
    request<PageResult<OpLog>>(`/api/v1/logs${qs(params ?? {})}`),
  loginLogs: (params?: { username?: string; status?: string; page?: number; pageSize?: number }) =>
    request<PageResult<LoginLog>>(`/api/v1/logs/login${qs(params ?? {})}`),
  apiLogs: (params?: { traceId?: string; path?: string; page?: number; pageSize?: number }) =>
    request<PageResult<APILog>>(`/api/v1/logs/api${qs(params ?? {})}`),
  clearLogs: (kind: "op" | "login" | "api" = "op") =>
    request<{ cleared: boolean }>(`/api/v1/logs?kind=${kind}`, { method: "DELETE" }),
  purgeLogs: (days = 30) =>
    request<{ purged: boolean; retentionDays: number }>(`/api/v1/logs/purge?days=${days}`, {
      method: "POST",
    }),
  exportLogs: (params?: { username?: string; module?: string; action?: string }) =>
    downloadCSV(`/api/v1/logs/export${qs(params ?? {})}`, "op-logs.csv"),
}
