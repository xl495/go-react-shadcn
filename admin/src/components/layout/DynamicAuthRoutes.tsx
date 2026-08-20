import { Suspense } from "react"
import { Route } from "react-router-dom"
import { PageFallback } from "@/components/PageFallback"
import { RequirePerm } from "@/providers/auth"
import { FALLBACK_MENU_ROUTES, flattenMenuRoutes, useMenus } from "@/hooks/queries"
import { resolvePage } from "@/constants/route-registry"
import { SettingsPage } from "@/pages/Settings"
import { UserDetailPage } from "@/pages/UserDetail"

function menuRouteProps(routePath: string): { index?: boolean; path?: string } {
  if (routePath === "/") return { index: true }
  return { path: routePath.replace(/^\//, "") }
}

function renderMenuRoute(m: (typeof FALLBACK_MENU_ROUTES)[number]) {
  const Page = resolvePage(m.component)
  if (!Page) return null
  const props = menuRouteProps(m.routePath)
  const element = (
    <RequirePerm perm={m.code}>
      <Suspense fallback={<PageFallback />}>
        <Page />
      </Suspense>
    </RequirePerm>
  )
  if (props.index) {
    return <Route key={m.code} index element={element} />
  }
  return <Route key={m.code} path={props.path} element={element} />
}

export function DynamicAuthRoutes() {
  const { data: menus, isLoading } = useMenus()
  const routes = flattenMenuRoutes(menus?.length ? menus : FALLBACK_MENU_ROUTES)

  if (isLoading && !menus) {
    return <Route index element={<PageFallback />} />
  }

  return (
    <>
      <Route path="settings" element={<SettingsPage />} />
      <Route
        path="users/:id"
        element={
          <RequirePerm perm="user:list">
            <UserDetailPage />
          </RequirePerm>
        }
      />
      {routes.map((m) => renderMenuRoute(m))}
    </>
  )
}
