import { request, qs } from "./http"
import type { PageResult, SysConfig } from "@/types"

export const configsApi = {
  configs: (params?: { page?: number; pageSize?: number; group?: string; q?: string }) =>
    request<PageResult<SysConfig>>(`/api/v1/configs${qs(params ?? {})}`),
  createConfig: (body: { key: string; value: string; name: string; group?: string; remark?: string }) =>
    request<SysConfig>("/api/v1/configs", { method: "POST", body: JSON.stringify(body) }),
  updateConfig: (id: number, body: { value?: string; name?: string; group?: string; remark?: string }) =>
    request<SysConfig>(`/api/v1/configs/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteConfig: (id: number) =>
    request<{ deleted: number }>(`/api/v1/configs/${id}`, { method: "DELETE" }),
  testMail: (to: string) =>
    request<{ sent: boolean; to: string }>("/api/v1/mail/test", {
      method: "POST",
      body: JSON.stringify({ to }),
    }),
}
