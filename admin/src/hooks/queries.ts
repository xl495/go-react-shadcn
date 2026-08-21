import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, isSessionExpired, getToken } from "@/api/client"
import type { MenuNode } from "@/types"

export const PAGE_SIZE = 10
export const PICKER_PAGE_SIZE = 200

export const FALLBACK_MENU_ROUTES: MenuNode[] = [
  { id: 0, code: "dashboard:read", name: "Dashboard", kind: "menu", routePath: "/", component: "DashboardPage", icon: "LayoutDashboard", sort: 10, hidden: false },
  { id: 0, code: "notify:list", name: "Notifications", kind: "menu", routePath: "/notifications", component: "NotificationsPage", icon: "Bell", sort: 12, hidden: false },
  { id: 0, code: "user:list", name: "Staff users", kind: "menu", routePath: "/users", component: "UsersPage", icon: "Users", sort: 20, hidden: false, permCode: "user:list" },
  { id: 0, code: "webuser:list", name: "Web users", kind: "menu", routePath: "/web-users", component: "WebUsersPage", icon: "Globe", sort: 21, hidden: false, permCode: "user:list" },
  { id: 0, code: "dept:list", name: "Departments", kind: "menu", routePath: "/departments", component: "DepartmentsPage", icon: "Building2", sort: 25, hidden: false },
  { id: 0, code: "role:list", name: "Roles", kind: "menu", routePath: "/roles", component: "RolesPage", icon: "Shield", sort: 30, hidden: false },
  { id: 0, code: "perm:list", name: "Permissions", kind: "menu", routePath: "/permissions", component: "PermissionsPage", icon: "KeyRound", sort: 40, hidden: false },
  { id: 0, code: "dict:list", name: "Dicts", kind: "menu", routePath: "/dicts", component: "DictsPage", icon: "BookMarked", sort: 50, hidden: false },
  { id: 0, code: "menu:list", name: "Menus", kind: "menu", routePath: "/menus", component: "MenusPage", icon: "PanelTop", sort: 55, hidden: false },
  { id: 0, code: "config:list", name: "Configs", kind: "menu", routePath: "/configs", component: "ConfigsPage", icon: "Settings2", sort: 60, hidden: false },
  { id: 0, code: "mail:jobs:list", name: "Mail queue", kind: "menu", routePath: "/mail/jobs", component: "MailJobsPage", icon: "Mail", sort: 65, hidden: false },
  { id: 0, code: "mail:campaign:list", name: "Templates", kind: "menu", routePath: "/mail/campaigns", component: "MailCampaignsPage", icon: "FileText", sort: 66, hidden: false },
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

function invalidate(qc: ReturnType<typeof useQueryClient>, ...keys: string[][]) {
  return Promise.all(keys.map((queryKey) => qc.invalidateQueries({ queryKey })))
}

function isAuthError(err: unknown) {
  return isSessionExpired(err)
}

export function useMenus() {
  return useQuery({
    queryKey: ["menus"],
    queryFn: () => api.menus(),
    enabled: !!getToken(),
    retry: (count, err) => count < 2 && !isAuthError(err),
  })
}

export function useDashboardStats() {
  return useQuery({
    queryKey: ["dashboard", "stats"],
    queryFn: () => api.stats(),
  })
}

export function useAuthSettings() {
  return useQuery({
    queryKey: ["auth", "settings"],
    queryFn: () => api.settings(),
    staleTime: 30_000,
  })
}

export function useLoginMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.login,
    onSuccess: (result) => {
      if (result.totpRequired || !result.token || !result.user) return
      qc.setQueryData(["auth", "me"], result.user)
      void invalidate(qc, ["menus"])
    },
  })
}

export function useGoogleAuthMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.google,
    onSuccess: (result) => {
      if (result.totpRequired || !result.token || !result.user) return
      qc.setQueryData(["auth", "me"], result.user)
      void invalidate(qc, ["menus"])
    },
  })
}

export function useMe(enabled: boolean) {
  return useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => api.me(),
    enabled,
    retry: false,
  })
}

export function useUpdateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.updateProfile,
    onSuccess: (user) => {
      qc.setQueryData(["auth", "me"], user)
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: api.changePassword,
  })
}

export function useUsers(params?: {
  page?: number
  pageSize?: number
  q?: string
  status?: string
  gender?: string
  department?: string
  roleId?: number
  kind?: string
}) {
  return useQuery({
    queryKey: ["users", "list", params],
    queryFn: () => api.users(params),
    placeholderData: keepPreviousData,
  })
}

export function useUser(id: number, kind?: string) {
  return useQuery({
    queryKey: ["users", "detail", id, kind],
    queryFn: () => api.getUser(id, kind),
    enabled: id > 0,
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createUser,
    onSuccess: (user) => {
      qc.setQueryData(["users", "detail", user.id, user.kind], user)
      void qc.invalidateQueries({ queryKey: ["users", "list"] })
    },
  })
}

export function useUpdateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateUser>[1] }) =>
      api.updateUser(id, body),
    onSuccess: (user) => {
      qc.setQueryData(["users", "detail", user.id, user.kind], user)
      void qc.invalidateQueries({ queryKey: ["users", "list"] })
    },
  })
}

export function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, kind }: { id: number; kind?: string }) => api.deleteUser(id, kind),
    onSuccess: (_d, vars) => {
      qc.removeQueries({ queryKey: ["users", "detail", vars.id, vars.kind] })
      void qc.invalidateQueries({ queryKey: ["users", "list"] })
    },
  })
}

export function useAssignUserRoles() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, roleIds, kind }: { id: number; roleIds: number[]; kind?: string }) =>
      api.assignUserRoles(id, roleIds, kind),
    onSuccess: (user) => {
      qc.setQueryData(["users", "detail", user.id, user.kind], user)
      void qc.invalidateQueries({ queryKey: ["users", "list"] })
    },
  })
}

export function useRevokeUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, kind }: { id: number; kind?: string }) => api.revokeUser(id, kind),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: ["users", "detail", vars.id] })
      void qc.invalidateQueries({ queryKey: ["users", "sessions", vars.id] })
    },
  })
}

export function useUserSessions(id: number, kind?: string) {
  return useQuery({
    queryKey: ["users", "sessions", id, kind],
    queryFn: () => api.userSessions(id, kind),
    enabled: id > 0,
  })
}

export function useRevokeUserSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, sid, kind }: { id: number; sid: number; kind?: string }) =>
      api.revokeUserSession(id, sid, kind),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: ["users", "sessions", vars.id] })
    },
  })
}

export function useImportUsers() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, kind }: { file: File; kind?: string }) => api.importUsers(file, kind),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["users", "list"] })
    },
  })
}

export function useRoles(params?: { page?: number; pageSize?: number; q?: string }, enabled = true) {
  return useQuery({
    queryKey: ["roles", params],
    queryFn: () => api.roles(params),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useRole(id: number) {
  return useQuery({
    queryKey: ["roles", "detail", id],
    queryFn: () => api.getRole(id),
    enabled: id > 0,
  })
}

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createRole,
    onSuccess: (role) => {
      qc.setQueryData(["roles", "detail", role.id], role)
      void invalidate(qc, ["roles"])
    },
  })
}

export function useUpdateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateRole>[1] }) =>
      api.updateRole(id, body),
    onSuccess: (role) => {
      qc.setQueryData(["roles", "detail", role.id], role)
      void invalidate(qc, ["roles"])
    },
  })
}

export function useDeleteRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteRole,
    onSuccess: () => void invalidate(qc, ["roles", "menus"]),
  })
}

export function useAssignRolePermissions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, permissionIds }: { id: number; permissionIds: number[] }) =>
      api.assignRolePermissions(id, permissionIds),
    onSuccess: (role) => {
      qc.setQueryData(["roles", "detail", role.id], role)
      void invalidate(qc, ["roles", "menus"])
    },
  })
}

export function usePermissions(params?: { page?: number; pageSize?: number; q?: string; kind?: string }) {
  return useQuery({
    queryKey: ["permissions", params],
    queryFn: () => api.permissions(params),
    placeholderData: keepPreviousData,
  })
}

export function useAllPermissions() {
  return useQuery({
    queryKey: ["permissions", "all"],
    queryFn: async () => {
      const pageSize = 200
      const first = await api.permissions({ page: 1, pageSize })
      const items = [...(first.items ?? [])]
      const total = first.total ?? items.length
      for (let page = 2; items.length < total; page++) {
        const next = await api.permissions({ page, pageSize })
        const batch = next.items ?? []
        if (batch.length === 0) break
        items.push(...batch)
      }
      return items
    },
  })
}

export function useCreatePermission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createPermission,
    onSuccess: () => void invalidate(qc, ["permissions", "menus"]),
  })
}

export function useDeletePermission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deletePermission,
    onSuccess: () => void invalidate(qc, ["permissions", "menus"]),
  })
}

export function useDicts(params?: { page?: number; pageSize?: number; q?: string }) {
  return useQuery({
    queryKey: ["dicts", params],
    queryFn: () => api.dicts(params),
    placeholderData: keepPreviousData,
  })
}

export function useDictItems(id: number, params?: { page?: number; pageSize?: number }) {
  return useQuery({
    queryKey: ["dict-items", id, params],
    queryFn: () => api.dictItems(id, params),
    enabled: id > 0,
  })
}

export function useCreateDict() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createDict,
    onSuccess: () => void invalidate(qc, ["dicts"]),
  })
}

export function useDeleteDict() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteDict,
    onSuccess: () => void invalidate(qc, ["dicts", "dict-items"]),
  })
}

export function useCreateDictItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.createDictItem>[1] }) =>
      api.createDictItem(id, body),
    onSuccess: () => void invalidate(qc, ["dict-items", "dicts"]),
  })
}

export function useDeleteDictItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteDictItem,
    onSuccess: () => void invalidate(qc, ["dict-items"]),
  })
}

export function useConfigs(params?: { page?: number; pageSize?: number; group?: string; q?: string }) {
  return useQuery({
    queryKey: ["configs", params],
    queryFn: () => api.configs(params),
    placeholderData: keepPreviousData,
  })
}

export function useCreateConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createConfig,
    onSuccess: () => void invalidate(qc, ["configs"]),
  })
}

export function useUpdateConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateConfig>[1] }) =>
      api.updateConfig(id, body),
    onSuccess: () => void invalidate(qc, ["configs"]),
  })
}

export function useSaveConfigs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (
      items: Array<
        | { id: number; body: Parameters<typeof api.updateConfig>[1] }
        | { create: Parameters<typeof api.createConfig>[0] }
      >,
    ) => {
      return api.batchConfigs(
        items.map((item) =>
          "create" in item
            ? item.create
            : { id: item.id, ...item.body },
        ),
      )
    },
    onSuccess: () => void invalidate(qc, ["configs"]),
  })
}

export function useDeleteConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteConfig,
    onSuccess: () => void invalidate(qc, ["configs"]),
  })
}

export function useTestMail() {
  return useMutation({
    mutationFn: (to: string) => api.testMail(to),
  })
}

export function useMailJobs(params?: { page?: number; pageSize?: number; status?: string; class?: string }) {
  return useQuery({
    queryKey: ["mail-jobs", params],
    queryFn: () => api.jobs(params),
    placeholderData: keepPreviousData,
    refetchInterval: (query) => {
      const items = query.state.data?.items ?? []
      return items.some((job) => job.status === "queued" || job.status === "sending") ? 4000 : false
    },
  })
}

export function useRetryMailJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.retryJob,
    onSuccess: () => void invalidate(qc, ["mail-jobs"]),
  })
}

export function useCancelMailJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.cancelJob,
    onSuccess: () => void invalidate(qc, ["mail-jobs"]),
  })
}

export function useMailCampaign(id: number) {
  return useQuery({
    queryKey: ["mail-campaigns", id],
    queryFn: () => api.getCampaign(id),
    enabled: id > 0,
  })
}

export function useMailCampaigns(params?: { page?: number; pageSize?: number; status?: string }) {
  return useQuery({
    queryKey: ["mail-campaigns", params],
    queryFn: () => api.campaigns(params),
    placeholderData: keepPreviousData,
  })
}

export function useCreateMailCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createCampaign,
    onSuccess: () => void invalidate(qc, ["mail-campaigns"]),
  })
}

export function useUpdateMailCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateCampaign>[1] }) =>
      api.updateCampaign(id, body),
    onSuccess: () => void invalidate(qc, ["mail-campaigns", "mail-jobs"]),
  })
}

export function useDeleteMailCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteCampaign,
    onSuccess: () => void invalidate(qc, ["mail-campaigns"]),
  })
}

export function useScheduleMailCampaign() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, scheduledAt }: { id: number; scheduledAt?: string }) =>
      api.scheduleCampaign(id, scheduledAt),
    onSuccess: () => void invalidate(qc, ["mail-campaigns", "mail-jobs"]),
  })
}

export function useDepartments(params?: { page?: number; pageSize?: number; q?: string }) {
  return useQuery({
    queryKey: ["departments", params],
    queryFn: () => api.departments(params),
    placeholderData: keepPreviousData,
  })
}

export function useCreateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createDepartment,
    onSuccess: () => void invalidate(qc, ["departments"], ["dicts"]),
  })
}

export function useUpdateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateDepartment>[1] }) =>
      api.updateDepartment(id, body),
    onSuccess: () => void invalidate(qc, ["departments"], ["dicts"]),
  })
}

export function useDeleteDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteDepartment,
    onSuccess: () => void invalidate(qc, ["departments"], ["dicts"]),
  })
}

export function useOpLogs(
  params?: {
    username?: string
    module?: string
    action?: string
    page?: number
    pageSize?: number
  },
  enabled = true,
) {
  return useQuery({
    queryKey: ["logs", "op", params],
    queryFn: () => api.logs(params),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useLoginLogs(
  params?: {
    username?: string
    status?: string
    page?: number
    pageSize?: number
  },
  enabled = true,
) {
  return useQuery({
    queryKey: ["logs", "login", params],
    queryFn: () => api.loginLogs(params),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useAPILogs(
  params?: { traceId?: string; path?: string; page?: number; pageSize?: number },
  enabled = true,
) {
  return useQuery({
    queryKey: ["logs", "api", params],
    queryFn: () => api.apiLogs(params),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useClearLogs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.clearLogs,
    onSuccess: (_d, kind) => {
      void qc.invalidateQueries({ queryKey: ["logs", kind === "login" ? "login" : kind === "api" ? "api" : "op"] })
    },
  })
}

export function usePurgeLogs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.purgeLogs,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["logs", "op"] })
      void qc.invalidateQueries({ queryKey: ["logs", "login"] })
      void qc.invalidateQueries({ queryKey: ["logs", "api"] })
    },
  })
}

export function useNavMenus(audience: string) {
  return useQuery({
    queryKey: ["nav-menus", audience],
    queryFn: () => api.navMenus(audience),
  })
}

export function useCreateNavMenu() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createNavMenu,
    onSuccess: () => void invalidate(qc, ["nav-menus"], ["menus"]),
  })
}

export function useUpdateNavMenu() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateNavMenu>[1] }) =>
      api.updateNavMenu(id, body),
    onSuccess: () => void invalidate(qc, ["nav-menus"], ["menus"]),
  })
}

export function useDeleteNavMenu() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteNavMenu,
    onSuccess: () => void invalidate(qc, ["nav-menus"], ["menus"]),
  })
}

export function useReorderNavMenus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.reorderNavMenus,
    onSuccess: () => void invalidate(qc, ["nav-menus"], ["menus"]),
  })
}

export function useNotifications(params?: { page?: number; pageSize?: number; unread?: boolean }) {
  return useQuery({
    queryKey: ["notifications", params],
    queryFn: () => api.notifications(params),
    placeholderData: keepPreviousData,
    refetchInterval: 30_000,
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ["notifications", "unread"],
    queryFn: () => api.unreadCount(),
    enabled: !!getToken(),
    refetchInterval: 30_000,
  })
}

export function useReadNotification() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.readNotification,
    onSuccess: () => void invalidate(qc, ["notifications"]),
  })
}

export function useReadAllNotifications() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.readAllNotifications,
    onSuccess: () => void invalidate(qc, ["notifications"]),
  })
}

export function useAnnounce() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.announce,
    onSuccess: () => void invalidate(qc, ["notifications"]),
  })
}
