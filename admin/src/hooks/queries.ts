import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, ApiError, getToken } from "@/api/client"
import type { MenuNode } from "@/types"

export const PAGE_SIZE = 10
export const PICKER_PAGE_SIZE = 200

export const FALLBACK_MENU_ROUTES: MenuNode[] = [
  { id: 0, code: "dashboard:read", name: "Dashboard", kind: "menu", routePath: "/", component: "DashboardPage", icon: "LayoutDashboard", sort: 10, hidden: false },
  { id: 0, code: "user:list", name: "Users", kind: "menu", routePath: "/users", component: "UsersPage", icon: "Users", sort: 20, hidden: false },
  { id: 0, code: "dept:list", name: "Departments", kind: "menu", routePath: "/departments", component: "DepartmentsPage", icon: "Building2", sort: 25, hidden: false },
  { id: 0, code: "role:list", name: "Roles", kind: "menu", routePath: "/roles", component: "RolesPage", icon: "Shield", sort: 30, hidden: false },
  { id: 0, code: "perm:list", name: "Permissions", kind: "menu", routePath: "/permissions", component: "PermissionsPage", icon: "KeyRound", sort: 40, hidden: false },
  { id: 0, code: "dict:list", name: "Dicts", kind: "menu", routePath: "/dicts", component: "DictsPage", icon: "BookMarked", sort: 50, hidden: false },
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
  return err instanceof ApiError && (err.status === 401 || err.code === 40101 || err.code === 40102)
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

export function useCaptcha() {
  return useQuery({
    queryKey: ["auth", "captcha"],
    queryFn: () => api.captcha(),
    staleTime: 0,
    refetchOnMount: "always",
  })
}

export function useLoginMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.login,
    onSuccess: (result) => {
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
}) {
  return useQuery({
    queryKey: ["users", params],
    queryFn: () => api.users(params),
    placeholderData: keepPreviousData,
  })
}

export function useUser(id: number) {
  return useQuery({
    queryKey: ["users", id],
    queryFn: () => api.getUser(id),
    enabled: id > 0,
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createUser,
    onSuccess: () => void invalidate(qc, ["users"]),
  })
}

export function useUpdateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateUser>[1] }) =>
      api.updateUser(id, body),
    onSuccess: () => void invalidate(qc, ["users"]),
  })
}

export function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteUser,
    onSuccess: () => void invalidate(qc, ["users"]),
  })
}

export function useAssignUserRoles() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, roleIds }: { id: number; roleIds: number[] }) => api.assignUserRoles(id, roleIds),
    onSuccess: () => void invalidate(qc, ["users"]),
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

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createRole,
    onSuccess: () => void invalidate(qc, ["roles"]),
  })
}

export function useUpdateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateRole>[1] }) =>
      api.updateRole(id, body),
    onSuccess: () => void invalidate(qc, ["roles"]),
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
    onSuccess: () => void invalidate(qc, ["roles", "menus"]),
  })
}

export function usePermissions(params?: { page?: number; pageSize?: number; q?: string; kind?: string }) {
  return useQuery({
    queryKey: ["permissions", params],
    queryFn: () => api.permissions(params),
    placeholderData: keepPreviousData,
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
    onSuccess: () => void invalidate(qc, ["departments"]),
  })
}

export function useUpdateDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: Parameters<typeof api.updateDepartment>[1] }) =>
      api.updateDepartment(id, body),
    onSuccess: () => void invalidate(qc, ["departments"]),
  })
}

export function useDeleteDepartment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteDepartment,
    onSuccess: () => void invalidate(qc, ["departments"]),
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

export function useAPILogs(params?: { traceId?: string; path?: string; page?: number; pageSize?: number }) {
  return useQuery({
    queryKey: ["logs", "api", params],
    queryFn: () => api.apiLogs(params),
  })
}

export function useClearLogs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.clearLogs,
    onSuccess: () => void invalidate(qc, ["logs"]),
  })
}

export function usePurgeLogs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.purgeLogs,
    onSuccess: () => void invalidate(qc, ["logs"]),
  })
}
