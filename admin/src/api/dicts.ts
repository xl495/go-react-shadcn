import { request, qs } from "./http"
import type { DictItem, DictLookup, DictType, PageResult } from "@/types"

export const dictsApi = {
  dicts: (params?: { page?: number; pageSize?: number }) =>
    request<PageResult<DictType>>(`/api/v1/dicts${qs(params ?? {})}`),
  createDict: (body: { code: string; name: string; status?: string; remark?: string }) =>
    request<DictType>("/api/v1/dicts", { method: "POST", body: JSON.stringify(body) }),
  updateDict: (id: number, body: { name?: string; status?: string; remark?: string }) =>
    request<DictType>(`/api/v1/dicts/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDict: (id: number) => request<{ deleted: number }>(`/api/v1/dicts/${id}`, { method: "DELETE" }),
  dictItems: (id: number, params?: { page?: number; pageSize?: number }) =>
    request<PageResult<DictItem>>(`/api/v1/dicts/${id}/items${qs(params ?? {})}`),
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
  lookupDict: (code: string) => request<DictLookup>(`/api/v1/dicts/by/${code}`),
}
