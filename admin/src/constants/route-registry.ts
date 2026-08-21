import { lazy, type ComponentType, type LazyExoticComponent } from "react"

export type PageKey =
  | "DashboardPage"
  | "UsersPage"
  | "WebUsersPage"
  | "DepartmentsPage"
  | "RolesPage"
  | "PermissionsPage"
  | "DictsPage"
  | "ConfigsPage"
  | "LogsPage"
  | "MailJobsPage"
  | "MailCampaignsPage"
  | "MenusPage"
  | "NotificationsPage"

export const PAGE_REGISTRY: Record<
  PageKey,
  LazyExoticComponent<ComponentType<object>>
> = {
  DashboardPage: lazy(() => import("@/pages/Dashboard").then((m) => ({ default: m.DashboardPage }))),
  UsersPage: lazy(() => import("@/pages/Users").then((m) => ({ default: m.UsersPage }))),
  WebUsersPage: lazy(() => import("@/pages/Users").then((m) => ({ default: m.WebUsersPage }))),
  DepartmentsPage: lazy(() =>
    import("@/pages/Departments").then((m) => ({ default: m.DepartmentsPage })),
  ),
  RolesPage: lazy(() => import("@/pages/Roles").then((m) => ({ default: m.RolesPage }))),
  PermissionsPage: lazy(() =>
    import("@/pages/Permissions").then((m) => ({ default: m.PermissionsPage })),
  ),
  DictsPage: lazy(() => import("@/pages/Dicts").then((m) => ({ default: m.DictsPage }))),
  ConfigsPage: lazy(() => import("@/pages/Configs").then((m) => ({ default: m.ConfigsPage }))),
  LogsPage: lazy(() => import("@/pages/Logs").then((m) => ({ default: m.LogsPage }))),
  MailJobsPage: lazy(() => import("@/pages/MailJobs").then((m) => ({ default: m.MailJobsPage }))),
  MailCampaignsPage: lazy(() =>
    import("@/pages/MailCampaigns").then((m) => ({ default: m.MailCampaignsPage })),
  ),
  MenusPage: lazy(() => import("@/pages/Menus").then((m) => ({ default: m.MenusPage }))),
  NotificationsPage: lazy(() =>
    import("@/pages/Notifications").then((m) => ({ default: m.NotificationsPage })),
  ),
}

export function resolvePage(component: string) {
  return PAGE_REGISTRY[component as PageKey]
}
