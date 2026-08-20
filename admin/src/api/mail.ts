import { request, qs } from "./http"
import type { MailCampaign, MailJob, PageResult } from "@/types"

export const mailApi = {
  jobs: (params?: { page?: number; pageSize?: number; status?: string; class?: string }) =>
    request<PageResult<MailJob>>(`/api/v1/mail/jobs${qs(params ?? {})}`),
  retryJob: (id: number) =>
    request<{ retried: boolean; id: number }>(`/api/v1/mail/jobs/${id}/retry`, { method: "POST" }),
  cancelJob: (id: number) =>
    request<{ canceled: boolean; id: number }>(`/api/v1/mail/jobs/${id}/cancel`, { method: "POST" }),
  getCampaign: (id: number) => request<MailCampaign>(`/api/v1/mail/campaigns/${id}`),
  campaigns: (params?: { page?: number; pageSize?: number; status?: string }) =>
    request<PageResult<MailCampaign>>(`/api/v1/mail/campaigns${qs(params ?? {})}`),
  createCampaign: (body: { name: string; subject: string; body: string; audience?: string }) =>
    request<MailCampaign>("/api/v1/mail/campaigns", { method: "POST", body: JSON.stringify(body) }),
  updateCampaign: (
    id: number,
    body: { name?: string; subject?: string; body?: string; audience?: string; status?: string },
  ) => request<MailCampaign>(`/api/v1/mail/campaigns/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteCampaign: (id: number) =>
    request<{ deleted: number }>(`/api/v1/mail/campaigns/${id}`, { method: "DELETE" }),
  scheduleCampaign: (id: number, scheduledAt?: string) =>
    request<MailCampaign>(`/api/v1/mail/campaigns/${id}/schedule`, {
      method: "POST",
      body: JSON.stringify(scheduledAt ? { scheduledAt } : {}),
    }),
  unsubscribe: (token: string) =>
    request<{ unsubscribed: boolean }>("/api/v1/mail/unsubscribe", {
      method: "POST",
      body: JSON.stringify({ token }),
    }),
}
