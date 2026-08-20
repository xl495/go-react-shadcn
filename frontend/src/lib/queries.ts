import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { MenuNode } from "@/lib/types"

/** Static fallback when menu API is empty (e.g. before seed migration). */
export const FALLBACK_MENU_ROUTES: MenuNode[] = [
  { id: 0, code: "dashboard:read", name: "Dashboard", kind: "menu", routePath: "/", component: "DashboardPage", icon: "LayoutDashboard", sort: 10, hidden: false },
  { id: 0, code: "user:list", name: "Users", kind: "menu", routePath: "/users", component: "UsersPage", icon: "Users", sort: 20, hidden: false },
  { id: 0, code: "role:list", name: "Roles", kind: "menu", routePath: "/roles", component: "RolesPage", icon: "Shield", sort: 30, hidden: false },
  { id: 0, code: "perm:list", name: "Permissions", kind: "menu", routePath: "/permissions", component: "PermissionsPage", icon: "KeyRound", sort: 40, hidden: false },
  { id: 0, code: "dict:list", name: "Dicts", kind: "menu", routePath: "/dicts", component: "DictsPage", icon: "BookMarked", sort: 50, hidden: false },
  { id: 0, code: "config:list", name: "Configs", kind: "menu", routePath: "/configs", component: "ConfigsPage", icon: "Settings2", sort: 60, hidden: false },
  { id: 0, code: "log:list", name: "Logs", kind: "menu", routePath: "/logs", component: "LogsPage", icon: "ClipboardList", sort: 70, hidden: false },
]

export function flattenMenuRoutes(nodes: MenuNode[]) {
  const out: MenuNode[] = []
  for (const n of nodes) {
    if (n.kind === "menu" && n.routePath && n.component && !n.hidden) out.push(n)
    if (n.children?.length) out.push(...flattenMenuRoutes(n.children))
  }
  return out.sort((a, b) => a.sort - b.sort)
}

export function useMenus() {
  return useQuery({
    queryKey: ["menus"],
    queryFn: () => api.menus(),
  })
}

export function useDashboardStats() {
  return useQuery({
    queryKey: ["dashboard", "stats"],
    queryFn: () => api.stats(),
  })
}

export function useUsers(params?: { page?: number; pageSize?: number; q?: string }) {
  return useQuery({
    queryKey: ["users", params],
    queryFn: () => api.users(params),
  })
}

export function useRoles(params?: { page?: number; pageSize?: number }, enabled = true) {
  return useQuery({
    queryKey: ["roles", params],
    queryFn: () => api.roles(params),
    enabled,
  })
}

export function usePermissions() {
  return useQuery({
    queryKey: ["permissions"],
    queryFn: () => api.permissions(),
  })
}

export function useOpLogs(params?: {
  username?: string
  module?: string
  action?: string
  page?: number
  pageSize?: number
}) {
  return useQuery({
    queryKey: ["logs", "op", params],
    queryFn: () => api.logs(params),
  })
}

export function useLoginLogs(params?: {
  username?: string
  status?: string
  page?: number
  pageSize?: number
}) {
  return useQuery({
    queryKey: ["logs", "login", params],
    queryFn: () => api.loginLogs(params),
  })
}

export function useAPILogs(params?: { traceId?: string; page?: number; pageSize?: number }) {
  return useQuery({
    queryKey: ["logs", "api", params],
    queryFn: () => api.apiLogs(params),
  })
}

export function useUser(id: number) {
  return useQuery({
    queryKey: ["users", id],
    queryFn: () => api.getUser(id),
    enabled: id > 0,
  })
}
