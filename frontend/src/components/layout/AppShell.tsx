import { useEffect, useRef, useState } from "react"
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import {
  BookMarked,
  ChevronDown,
  ClipboardList,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Settings,
  Settings2,
  Shield,
  Users,
} from "lucide-react"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { Avatar } from "@/components/ui/avatar"
import { useAuth } from "@/lib/auth"
import { roleLabel, useI18n } from "@/lib/i18n"
import { P } from "@/lib/perms"
import { cn } from "@/lib/utils"

export function AppShell() {
  const { user, logout, can } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const links = [
    { to: "/", label: t("nav.dashboard"), icon: LayoutDashboard, perm: P.dashboard },
    { to: "/users", label: t("nav.users"), icon: Users, perm: P.userList },
    { to: "/roles", label: t("nav.roles"), icon: Shield, perm: P.roleList },
    { to: "/permissions", label: t("nav.permissions"), icon: KeyRound, perm: P.permList },
    { to: "/dicts", label: t("nav.dicts"), icon: BookMarked, perm: P.dictList },
    { to: "/configs", label: t("nav.configs"), icon: Settings2, perm: P.configList },
    { to: "/logs", label: t("nav.logs"), icon: ClipboardList, perm: P.logList },
  ]
  const visible = links.filter((l) => can(l.perm))
  const current =
    visible.find((l) => (l.to === "/" ? location.pathname === "/" : location.pathname.startsWith(l.to)))
      ?.label ??
    (location.pathname.startsWith("/settings") ? t("nav.settings") : t("nav.dashboard"))
  const displayName = user?.nickname || user?.username || ""

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="flex w-56 shrink-0 flex-col border-r bg-sidebar">
        <div className="flex h-14 items-center border-b px-5">
          <span className="text-sm font-semibold tracking-tight">Latch</span>
        </div>
        <nav className="flex flex-1 flex-col gap-0.5 p-2">
          {visible.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.to === "/"}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
                  isActive
                    ? "bg-foreground text-background"
                    : "text-muted-foreground hover:bg-sidebar-accent hover:text-foreground",
                )
              }
            >
              <l.icon className="size-4" />
              {l.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b bg-background px-6">
          <div>
            <p className="text-[11px] text-muted-foreground">{t("app.console")}</p>
            <h1 className="text-sm font-medium">{current}</h1>
          </div>
          <div className="flex items-center gap-3">
            <LanguageSwitcher />
            <UserMenu
              name={displayName}
              avatar={user?.avatar}
              subtitle={(user?.roles ?? []).map((r) => roleLabel(r.code, r.name, t)).join(" · ")}
              onProfile={() => navigate("/settings")}
              onLogout={() => {
                logout()
                navigate("/login")
              }}
            />
          </div>
        </header>
        <main className="min-w-0 flex-1 overflow-auto bg-muted/40 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function UserMenu({
  name,
  avatar,
  subtitle,
  onProfile,
  onLogout,
}: {
  name: string
  avatar?: string
  subtitle: string
  onProfile: () => void
  onLogout: () => void
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener("mousedown", onDoc)
    return () => document.removeEventListener("mousedown", onDoc)
  }, [])

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 rounded-md px-1.5 py-1 hover:bg-accent"
      >
        <Avatar name={name} src={avatar} />
        <span className="hidden text-left text-sm sm:block">
          <span className="block leading-tight font-medium">{name}</span>
          <span className="block text-[11px] text-muted-foreground">{subtitle || t("app.noRole")}</span>
        </span>
        <ChevronDown className="size-3.5 text-muted-foreground" />
      </button>
      {open ? (
        <div className="absolute right-0 z-50 mt-1 w-48 rounded-md border bg-popover py-1 shadow-md">
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent"
            onClick={() => {
              setOpen(false)
              onProfile()
            }}
          >
            <Settings className="size-4" />
            {t("nav.settings")}
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent"
            onClick={() => {
              setOpen(false)
              onLogout()
            }}
          >
            <LogOut className="size-4" />
            {t("app.logout")}
          </button>
        </div>
      ) : null}
    </div>
  )
}
