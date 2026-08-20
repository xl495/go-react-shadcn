import { useEffect, useMemo, useState } from "react"
import { NavLink, Outlet, useNavigate } from "react-router-dom"
import { House, KeyRound, LogOut, PanelLeftClose, PanelLeftOpen, User } from "lucide-react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useResizableSidebar } from "@/lib/use-resizable-sidebar"
import type { MenuNode } from "@/lib/types"

const ICONS: Record<string, typeof House> = {
  House,
  User,
  KeyRound,
}

type NavItem = {
  key: string
  label: string
  to: string
  icon: typeof House
}

function navFromMenus(nodes: MenuNode[]): NavItem[] {
  const out: NavItem[] = []
  for (const n of nodes) {
    if (n.kind !== "menu" || n.hidden) continue
    if (n.routePath) {
      out.push({
        key: n.code,
        label: n.name,
        to: n.routePath,
        icon: ICONS[n.icon] ?? House,
      })
    }
    if (n.children?.length) {
      out.push(...navFromMenus(n.children))
    }
  }
  return out
}

export function AppShell() {
  const { user, setUser, logout } = useAuth()
  const navigate = useNavigate()
  const [menus, setMenus] = useState<MenuNode[]>([])
  const [error, setError] = useState("")

  useEffect(() => {
    let cancelled = false
    Promise.all([api.me(), api.webMenus()])
      .then(([me, tree]) => {
        if (cancelled) return
        setUser(me)
        setMenus(tree)
        setError("")
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof ApiError && (err.status === 401 || err.code === 40101 || err.code === 40102)) {
          logout()
          navigate("/login", { replace: true })
          return
        }
        setError(err instanceof ApiError ? err.message : "加载失败")
      })
    return () => {
      cancelled = true
    }
  }, [logout, navigate, setUser])

  const items = useMemo(() => navFromMenus(menus), [menus])
  const sidebar = useResizableSidebar("latch.web.sidebar.width")

  function onSignOut() {
    logout()
    navigate("/login", { replace: true })
  }

  return (
    <div className="flex h-full overflow-hidden bg-background text-foreground">
      <aside
        style={{ width: sidebar.width }}
        className={`relative flex h-full shrink-0 flex-col overflow-x-hidden border-r ${
          sidebar.resizing
            ? ""
            : "transition-[width] duration-200 ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none"
        }`}
      >
        <div className="flex h-14 shrink-0 items-center border-b px-4">
          <span className="truncate text-sm font-semibold tracking-tight">Latch</span>
        </div>
        <nav aria-label="主导航" className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto p-3">
          {items.map((item) => {
            const Icon = item.icon
            return (
              <NavLink
                key={item.key}
                to={item.to}
                end={item.to === "/"}
                title={item.label}
                className={({ isActive }) =>
                  `inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm ${
                    isActive ? "bg-muted font-medium" : "hover:bg-muted"
                  }`
                }
              >
                <Icon className="size-4 shrink-0" />
                <span className="min-w-0 truncate">{item.label}</span>
              </NavLink>
            )
          })}
        </nav>
        <div
          role="separator"
          aria-orientation="vertical"
          aria-valuemin={64}
          aria-valuemax={420}
          aria-valuenow={sidebar.width}
          aria-label="拖动调整侧栏宽度"
          tabIndex={0}
          className="absolute inset-y-0 right-0 z-10 w-2 cursor-col-resize touch-none bg-transparent after:absolute after:inset-y-0 after:right-0 after:w-px after:bg-transparent hover:after:bg-foreground/30 focus-visible:bg-foreground/10 focus-visible:outline-none"
          onPointerDown={sidebar.onResizePointerDown}
          onKeyDown={sidebar.onResizeKeyDown}
          onDoubleClick={sidebar.resetWidth}
        />
      </aside>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between border-b pr-6 pl-2">
          <button
            type="button"
            aria-label={sidebar.compact ? "展开侧栏" : "收起侧栏"}
            onClick={sidebar.toggleCollapsed}
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-md hover:bg-muted"
          >
            {sidebar.compact ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
          </button>
          <button
            type="button"
            onClick={onSignOut}
            className="inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm hover:bg-muted"
          >
            <LogOut className="size-4" />
            退出登录
          </button>
        </header>
        <main className="mx-auto flex min-h-0 w-full max-w-lg flex-1 flex-col overflow-auto overscroll-contain px-4 py-10">
          {error ? <p className="mb-4 text-sm text-destructive">{error}</p> : null}
          {user ? <Outlet /> : <p className="text-sm text-muted-foreground">加载中…</p>}
        </main>
      </div>
    </div>
  )
}
