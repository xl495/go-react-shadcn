import { Link, useLocation } from "react-router-dom"
import { useI18n } from "@/lib/i18n"
import { cn } from "@/lib/utils"

type Crumb = { label: string; to?: string }

export function Breadcrumbs() {
  const { pathname } = useLocation()
  const { t } = useI18n()
  const crumbs = crumbsFor(pathname, t)

  return (
    <nav aria-label="breadcrumb" className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
      {crumbs.map((c, i) => {
        const last = i === crumbs.length - 1
        return (
          <span key={`${c.label}-${i}`} className="flex items-center gap-1">
            {i > 0 ? <span>/</span> : null}
            {c.to && !last ? (
              <Link to={c.to} className="hover:text-foreground">
                {c.label}
              </Link>
            ) : (
              <span className={cn(last && "text-foreground")}>{c.label}</span>
            )}
          </span>
        )
      })}
    </nav>
  )
}

function crumbsFor(pathname: string, t: (k: string) => string): Crumb[] {
  const parts = pathname.split("/").filter(Boolean)
  const crumbs: Crumb[] = [{ label: t("nav.dashboard"), to: "/" }]
  if (parts.length === 0) return crumbs

  const head = parts[0]
  const system = new Set(["dicts", "configs", "logs"])
  if (system.has(head)) {
    crumbs.push({ label: t("nav.system") })
    const label =
      head === "dicts" ? t("nav.dicts") : head === "configs" ? t("nav.configs") : t("nav.logs")
    crumbs.push({ label, to: `/${head}` })
    return crumbs
  }

  if (head === "users") {
    crumbs.push({ label: t("nav.users"), to: "/users" })
    if (parts[1]) crumbs.push({ label: t("users.detail") })
    return crumbs
  }
  if (head === "roles") {
    crumbs.push({ label: t("nav.roles"), to: "/roles" })
    return crumbs
  }
  if (head === "permissions") {
    crumbs.push({ label: t("nav.permissions"), to: "/permissions" })
    return crumbs
  }
  if (head === "settings") {
    crumbs.push({ label: t("nav.settings"), to: "/settings" })
    return crumbs
  }
  return crumbs
}
