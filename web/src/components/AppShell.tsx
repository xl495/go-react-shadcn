import { useEffect, useMemo, useState } from "react"
import { NavLink, Outlet, useNavigate } from "react-router-dom"
import { CircleHelp, House, KeyRound, LogOut, PanelLeftClose, PanelLeftOpen, User, Bell } from "lucide-react"
import { api, ApiError, isSessionExpired } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { menuLabel, useI18n } from "@/lib/i18n"
import { useResizableSidebar } from "@/lib/use-resizable-sidebar"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import { ThemeToggle } from "@/components/ThemeToggle"
import type { MenuNode } from "@/lib/types"

const ICONS: Record<string, typeof House> = {
  House,
  User,
  KeyRound,
  Bell,
}

function menuIcon(name: string): typeof House {
  const Icon = ICONS[name]
  if (Icon) return Icon
  if (name) console.warn(`unknown menu icon: ${name}`)
  return CircleHelp
}

type NavItem = {
  key: string
  label: string
  to: string
  icon: typeof House
}

function navFromMenus(nodes: MenuNode[], t: (key: string) => string): NavItem[] {
  const out: NavItem[] = []
  for (const n of nodes) {
    if (n.kind !== "menu" || n.hidden) continue
    if (n.routePath) {
      out.push({
        key: n.code,
        label: menuLabel(n.code, n.name, t),
        to: n.routePath,
        icon: menuIcon(n.icon),
      })
    }
    if (n.children?.length) {
      out.push(...navFromMenus(n.children, t))
    }
  }
  return out
}

export function AppShell() {
  const { t } = useI18n()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [menus, setMenus] = useState<MenuNode[]>([])
  const [error, setError] = useState("")

  const [unread, setUnread] = useState(0)

  useEffect(() => {
    let cancelled = false
    api
      .webMenus()
      .then((tree) => {
        if (cancelled) return
        setMenus(tree)
        setError("")
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof ApiError && isSessionExpired(err)) {
          void logout().then(() => navigate("/login", { replace: true }))
          return
        }
        setError(err instanceof ApiError ? err.message : t("app.loadFailed"))
      })
    return () => {
      cancelled = true
    }
  }, [logout, navigate, t])

  useEffect(() => {
    let cancelled = false
    function tick() {
      api
        .unreadCount()
        .then((row) => {
          if (!cancelled) setUnread(row.unread)
        })
        .catch(() => undefined)
    }
    tick()
    const timer = window.setInterval(tick, 30_000)
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [])

  const items = useMemo(() => navFromMenus(menus, t), [menus, t])
  const sidebar = useResizableSidebar("latch.web.sidebar.width")

  function onSignOut() {
    void logout().then(() => navigate("/login", { replace: true }))
  }

  return (
    <div className="flex h-full overflow-hidden bg-background text-foreground">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-3 focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:shadow"
      >
        {t("nav.skipToContent")}
      </a>
      <aside
        style={{ width: sidebar.width }}
        className={`relative flex h-full shrink-0 flex-col overflow-x-hidden border-r ${
          sidebar.resizing
            ? ""
            : "transition-[width] duration-200 ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none"
        }`}
      >
        <div className="flex h-14 shrink-0 items-center gap-2 border-b px-3">
          <img src="/gra-mark.png" alt="" width={28} height={28} className="size-7 rounded-md" />
          <span className="font-display text-xl leading-none tracking-tight">gra</span>
        </div>
        <nav aria-label={t("nav.main")} className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto p-3">
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
          aria-label={t("nav.resize")}
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
            aria-label={sidebar.compact ? t("nav.expand") : t("nav.collapse")}
            onClick={sidebar.toggleCollapsed}
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-md hover:bg-muted"
          >
            {sidebar.compact ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
          </button>
          <div className="flex items-center gap-2">
            <NavLink
              to="/notifications"
              aria-label={t("nav.notifications")}
              className="relative inline-flex size-9 items-center justify-center rounded-md hover:bg-muted"
            >
              <Bell className="size-4" />
              {unread > 0 ? (
                <span className="absolute top-1 right-1 min-w-4 rounded-full bg-destructive px-1 text-[10px] leading-4 text-destructive-foreground">
                  {unread > 99 ? "99+" : unread}
                </span>
              ) : null}
            </NavLink>
            <ThemeToggle />
            <LanguageSwitcher />
            <button
              type="button"
              onClick={onSignOut}
              className="inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm hover:bg-muted"
            >
              <LogOut className="size-4" />
              {t("nav.signOut")}
            </button>
          </div>
        </header>
        <main id="main" className="mx-auto flex min-h-0 w-full max-w-lg flex-1 flex-col overflow-auto overscroll-contain px-4 py-10">
          {error ? <p className="mb-4 text-sm text-destructive">{error}</p> : null}
          {user ? <Outlet /> : <p className="text-sm text-muted-foreground">{t("app.loading")}</p>}
        </main>
      </div>
    </div>
  )
}
