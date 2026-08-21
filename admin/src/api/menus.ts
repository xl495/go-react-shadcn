import { request } from "./http"
import type { PageResult } from "@/types"

export type NavMenu = {
  id: number
  parentId?: number
  audience: string
  name: string
  code: string
  routePath: string
  component: string
  icon: string
  sort: number
  hidden: boolean
  permCode: string
  status: string
  isSystem: boolean
  children?: NavMenu[]
}

export type NavMenuInput = {
  parentId?: number | null
  audience: string
  name: string
  code: string
  routePath?: string
  component?: string
  icon?: string
  sort?: number
  hidden?: boolean
  permCode?: string
  status?: string
}

export const menusApi = {
  navMenus: (audience: string) => request<NavMenu[]>(`/api/v1/nav-menus?audience=${encodeURIComponent(audience)}`),
  createNavMenu: (body: NavMenuInput) =>
    request<NavMenu>("/api/v1/nav-menus", { method: "POST", body: JSON.stringify(body) }),
  updateNavMenu: (id: number, body: Partial<NavMenuInput>) =>
    request<NavMenu>(`/api/v1/nav-menus/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteNavMenu: (id: number) => request<{ deleted: number }>(`/api/v1/nav-menus/${id}`, { method: "DELETE" }),
  reorderNavMenus: (items: { id: number; sort: number; parentId?: number | null }[]) =>
    request<{ updated: number }>("/api/v1/nav-menus/reorder", { method: "PUT", body: JSON.stringify({ items }) }),
}

export type NotificationItem = {
  id: number
  type: string
  title: string
  body: string
  refType: string
  refId: number
  readAt?: string | null
  createdAt: string
}

export const notificationsApi = {
  notifications: (params?: { page?: number; pageSize?: number; unread?: boolean }) => {
    const q = new URLSearchParams()
    if (params?.page) q.set("page", String(params.page))
    if (params?.pageSize) q.set("pageSize", String(params.pageSize))
    if (params?.unread) q.set("unread", "1")
    const suffix = q.toString() ? `?${q.toString()}` : ""
    return request<PageResult<NotificationItem>>(`/api/v1/notifications${suffix}`)
  },
  unreadCount: () => request<{ unread: number }>("/api/v1/notifications/unread-count"),
  readNotification: (id: number) =>
    request<NotificationItem>(`/api/v1/notifications/${id}/read`, { method: "POST" }),
  readAllNotifications: () =>
    request<{ updated: number }>("/api/v1/notifications/read-all", { method: "POST" }),
  announce: (body: { kind: string; title: string; body?: string }) =>
    request<{ sent: number }>("/api/v1/announcements", { method: "POST", body: JSON.stringify(body) }),
}
