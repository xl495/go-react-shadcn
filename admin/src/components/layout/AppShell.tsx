import { useEffect, useState } from "react"
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import {
  BookMarked,
  Building2,
  ChevronDown,
  ClipboardList,
  FileText,
  FolderTree,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Mail,
  Monitor,
  Settings,
  Settings2,
  Shield,
  Users,
} from "lucide-react"
import { Breadcrumbs } from "@/components/layout/Breadcrumbs"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { Avatar } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useAuth } from "@/providers/auth"
import { roleLabel, useI18n } from "@/providers/i18n"
import { P } from "@/constants/perms"
import { useMenus } from "@/hooks/queries"
import { cn } from "@/utils/cn"
import type { MenuNode } from "@/types"

const ICONS: Record<string, typeof LayoutDashboard> = {
  LayoutDashboard,
  Users,
  Building2,
  Shield,
  KeyRound,
  BookMarked,
  Settings2,
  ClipboardList,
  Mail,
  FileText,
  FolderTree,
  Monitor,
}

type SidebarItem = {
  key: string
  label: string
  icon: typeof LayoutDashboard
  to?: string
  perm?: string
  children?: SidebarItem[]
}

function sidebarFromMenus(nodes: MenuNode[]): SidebarItem[] {
  const out: SidebarItem[] = []
  for (const n of nodes) {
    if (n.kind !== "menu" || n.hidden) continue
    const children = sidebarFromMenus(n.children ?? [])
    if (n.routePath) {
      out.push({
        key: n.code,
        label: n.name,
        icon: ICONS[n.icon] ?? LayoutDashboard,
        to: n.routePath,
        perm: n.code,
      })
      continue
    }
    if (children.length > 0) {
      out.push({
        key: n.code,
        label: n.name,
        icon: ICONS[n.icon] ?? FolderTree,
        children,
      })
    }
  }
  return out
}

function fallbackSidebar(t: (key: string) => string): SidebarItem[] {
  return [
    { key: P.dashboard, label: t("nav.dashboard"), icon: LayoutDashboard, to: "/", perm: P.dashboard },
    {
      key: "org",
      label: t("nav.org"),
      icon: FolderTree,
      children: [
        { key: P.userList, label: t("nav.users"), icon: Users, to: "/users", perm: P.userList },
        { key: P.deptList, label: t("nav.departments"), icon: Building2, to: "/departments", perm: P.deptList },
        { key: P.roleList, label: t("nav.roles"), icon: Shield, to: "/roles", perm: P.roleList },
        { key: P.permList, label: t("nav.permissions"), icon: KeyRound, to: "/permissions", perm: P.permList },
      ],
    },
    {
      key: "system",
      label: t("nav.system"),
      icon: Monitor,
      children: [
        { key: P.dictList, label: t("nav.dicts"), icon: BookMarked, to: "/dicts", perm: P.dictList },
        { key: P.configList, label: t("nav.configs"), icon: Settings2, to: "/configs", perm: P.configList },
        { key: P.mailJobsList, label: t("nav.mailJobs"), icon: Mail, to: "/mail/jobs", perm: P.mailJobsList },
        { key: P.mailCampaignList, label: t("nav.mailCampaigns"), icon: FileText, to: "/mail/campaigns", perm: P.mailCampaignList },
        { key: P.logList, label: t("nav.logs"), icon: ClipboardList, to: "/logs", perm: P.logList },
      ],
    },
  ]
}

function filterSidebar(items: SidebarItem[], can: (code: string) => boolean): SidebarItem[] {
  const out: SidebarItem[] = []
  for (const item of items) {
    if (item.children?.length) {
      const children = filterSidebar(item.children, can)
      if (children.length) out.push({ ...item, children })
      continue
    }
    if (!item.perm || can(item.perm)) out.push(item)
  }
  return out
}

function pathMatches(to: string, pathname: string) {
  return to === "/" ? pathname === "/" : pathname === to || pathname.startsWith(`${to}/`)
}

function isPathInGroup(item: SidebarItem, pathname: string): boolean {
  if (item.to) return pathMatches(item.to, pathname)
  return (item.children ?? []).some((child) => isPathInGroup(child, pathname))
}

function findCurrentLabel(items: SidebarItem[], pathname: string): string | undefined {
  let best: { label: string; len: number } | undefined
  function walk(nodes: SidebarItem[]) {
    for (const item of nodes) {
      if (item.to && pathMatches(item.to, pathname)) {
        if (!best || item.to.length > best.len) best = { label: item.label, len: item.to.length }
      }
      if (item.children) walk(item.children)
    }
  }
  walk(items)
  return best?.label
}

export function AppShell() {
  const { user, logout, can } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { data: menus = [] } = useMenus()

  const source = menus.length > 0 ? sidebarFromMenus(menus) : fallbackSidebar(t)
  const visible = filterSidebar(source, can)
  const current =
    location.pathname === "/mail/campaigns/new"
      ? t("mail.createCampaign")
      : /^\/mail\/campaigns\/\d+/.test(location.pathname)
        ? t("mail.editCampaign")
        : findCurrentLabel(visible, location.pathname) ??
          (location.pathname.startsWith("/settings/password")
            ? t("nav.password")
            : location.pathname.startsWith("/settings")
              ? t("nav.settings")
              : t("nav.dashboard"))
  const displayName = user?.nickname || user?.username || ""

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="flex w-56 shrink-0 flex-col border-r bg-sidebar">
        <div className="flex h-14 items-center border-b px-5">
          <span className="text-sm font-semibold tracking-tight">Latch</span>
        </div>
        <nav className="flex flex-1 flex-col gap-0.5 p-2">
          {visible.map((item) =>
            item.children?.length ? (
              <NavGroup key={item.key} item={item} pathname={location.pathname} />
            ) : (
              <NavItem key={item.key} to={item.to!} label={item.label} icon={item.icon} />
            ),
          )}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b bg-background px-6">
          <div>
            <Breadcrumbs />
            <h1 className="text-sm font-medium">{current}</h1>
          </div>
          <div className="flex items-center gap-3">
            <LanguageSwitcher />
            <UserMenu
              name={displayName}
              avatar={user?.avatar}
              subtitle={(user?.roles ?? []).map((r) => roleLabel(r.code, r.name, t)).join(" · ")}
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

function NavGroup({ item, pathname }: { item: SidebarItem; pathname: string }) {
  const Icon = item.icon
  const containsActive = isPathInGroup(item, pathname)
  const [open, setOpen] = useState(containsActive)
  useEffect(() => {
    if (containsActive) setOpen(true)
  }, [containsActive])

  return (
    <div>
      <Button
        type="button"
        variant="ghost"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className={cn(
          "h-auto w-full justify-start gap-2 px-3 py-2 font-normal",
          containsActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        <Icon className="size-4 shrink-0" />
        <span className="min-w-0 flex-1 truncate text-left">{item.label}</span>
        <ChevronDown className={cn("size-3.5 shrink-0 transition-transform", !open && "-rotate-90")} />
      </Button>
      {open ? (
        <div className="ml-3 border-l border-border/70 pl-1">
          {item.children!.map((child) =>
            child.children?.length ? (
              <NavGroup key={child.key} item={child} pathname={pathname} />
            ) : (
              <NavItem key={child.key} to={child.to!} label={child.label} icon={child.icon} />
            ),
          )}
        </div>
      ) : null}
    </div>
  )
}

function NavItem({
  to,
  label,
  icon: Icon,
}: {
  to: string
  label: string
  icon: typeof Users
}) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) =>
        cn(
          "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
          isActive
            ? "bg-foreground text-background"
            : "text-muted-foreground hover:bg-sidebar-accent hover:text-foreground",
        )
      }
    >
      <Icon className="size-4" />
      {label}
    </NavLink>
  )
}

function UserMenu({
  name,
  avatar,
  subtitle,
  onLogout,
}: {
  name: string
  avatar?: string
  subtitle: string
  onLogout: () => void
}) {
  const { t } = useI18n()
  const navigate = useNavigate()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="ghost" className="h-auto gap-2 px-1.5 py-1">
          <Avatar name={name} src={avatar} />
          <span className="hidden text-left text-sm sm:block">
            <span className="block leading-tight font-medium">{name}</span>
            <span className="block text-[11px] text-muted-foreground">{subtitle || t("app.noRole")}</span>
          </span>
          <ChevronDown className="size-3.5 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuItem onClick={() => navigate("/settings")}>
          <Settings />
          {t("nav.settings")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => navigate("/settings/password")}>
          <KeyRound />
          {t("nav.password")}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onLogout}>
          <LogOut />
          {t("app.logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
