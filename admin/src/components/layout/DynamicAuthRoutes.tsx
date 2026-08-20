import { lazy, Suspense } from "react"
import type { RouteObject } from "react-router-dom"
import { PageFallback } from "@/components/PageFallback"
import { RequirePerm } from "@/providers/auth"
import { FALLBACK_MENU_ROUTES, flattenMenuRoutes, useMenus } from "@/hooks/queries"
import { resolvePage } from "@/constants/route-registry"
import type { MenuNode } from "@/types"

const SettingsPage = lazy(() => import("@/pages/Settings").then((m) => ({ default: m.SettingsPage })))
const ChangePasswordPage = lazy(() =>
  import("@/pages/ChangePassword").then((m) => ({ default: m.ChangePasswordPage })),
)
const MailTemplateEditorPage = lazy(() =>
  import("@/pages/MailTemplateEditor").then((m) => ({ default: m.MailTemplateEditorPage })),
)
const UserDetailPage = lazy(() => import("@/pages/UserDetail").then((m) => ({ default: m.UserDetailPage })))

function menuRouteObject(m: MenuNode): RouteObject | null {
  const Page = resolvePage(m.component)
  if (!Page) return null
  const element = (
    <RequirePerm perm={m.code}>
      <Suspense fallback={<PageFallback />}>
        <Page />
      </Suspense>
    </RequirePerm>
  )
  if (m.routePath === "/") return { index: true, element }
  return { path: m.routePath.replace(/^\//, ""), element }
}

export function useDynamicAuthRoutes(): RouteObject[] {
  const { data: menus, isLoading } = useMenus()
  const routes = flattenMenuRoutes(menus?.length ? menus : FALLBACK_MENU_ROUTES)

  if (isLoading && !menus) {
    return [{ index: true, element: <PageFallback /> }]
  }

  return [
    {
      path: "settings/password",
      element: (
        <Suspense fallback={<PageFallback />}>
          <ChangePasswordPage />
        </Suspense>
      ),
    },
    {
      path: "settings",
      element: (
        <Suspense fallback={<PageFallback />}>
          <SettingsPage />
        </Suspense>
      ),
    },
    {
      path: "users/:id",
      element: (
        <RequirePerm perm="user:list">
          <Suspense fallback={<PageFallback />}>
            <UserDetailPage />
          </Suspense>
        </RequirePerm>
      ),
    },
    {
      path: "mail/campaigns/new",
      element: (
        <RequirePerm perm="mail:campaign:list">
          <Suspense fallback={<PageFallback />}>
            <MailTemplateEditorPage />
          </Suspense>
        </RequirePerm>
      ),
    },
    {
      path: "mail/campaigns/:id",
      element: (
        <RequirePerm perm="mail:campaign:list">
          <Suspense fallback={<PageFallback />}>
            <MailTemplateEditorPage />
          </Suspense>
        </RequirePerm>
      ),
    },
    ...routes.map(menuRouteObject).filter((r): r is RouteObject => r != null),
  ]
}
