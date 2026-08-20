import { lazy, type ComponentType, type LazyExoticComponent } from "react"

export type PageKey =
  | "DashboardPage"
  | "UsersPage"
  | "RolesPage"
  | "PermissionsPage"
  | "DictsPage"
  | "ConfigsPage"
  | "LogsPage"

export const PAGE_REGISTRY: Record<
  PageKey,
  LazyExoticComponent<ComponentType<object>>
> = {
  DashboardPage: lazy(() => import("@/pages/Dashboard").then((m) => ({ default: m.DashboardPage }))),
  UsersPage: lazy(() => import("@/pages/Users").then((m) => ({ default: m.UsersPage }))),
  RolesPage: lazy(() => import("@/pages/Roles").then((m) => ({ default: m.RolesPage }))),
  PermissionsPage: lazy(() =>
    import("@/pages/Permissions").then((m) => ({ default: m.PermissionsPage })),
  ),
  DictsPage: lazy(() => import("@/pages/Dicts").then((m) => ({ default: m.DictsPage }))),
  ConfigsPage: lazy(() => import("@/pages/Configs").then((m) => ({ default: m.ConfigsPage }))),
  LogsPage: lazy(() => import("@/pages/Logs").then((m) => ({ default: m.LogsPage }))),
}

export function resolvePage(component: string) {
  return PAGE_REGISTRY[component as PageKey]
}
