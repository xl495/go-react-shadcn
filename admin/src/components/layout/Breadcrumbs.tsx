import { Link, useLocation } from "react-router-dom"
import { useI18n } from "@/providers/i18n"
import { cn } from "@/utils/cn"

type Crumb = { label: string; to?: string }

export function Breadcrumbs() {
  const { pathname } = useLocation()
  const { t } = useI18n()
  const crumbs = crumbsFor(pathname, t)

  return (
    <nav aria-label={t("nav.breadcrumb")} className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
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
  const org = new Set(["users", "web-users", "departments", "roles", "permissions"])
  const system = new Set(["dicts", "configs", "logs", "mail"])
  if (org.has(head)) {
    crumbs.push({ label: t("nav.org") })
    if (head === "users") {
      crumbs.push({ label: t("nav.users"), to: "/users" })
      if (parts[1]) crumbs.push({ label: t("users.detail") })
      return crumbs
    }
    if (head === "web-users") {
      crumbs.push({ label: t("nav.webUsers"), to: "/web-users" })
      if (parts[1]) crumbs.push({ label: t("users.detail") })
      return crumbs
    }
    if (head === "departments") {
      crumbs.push({ label: t("nav.departments"), to: "/departments" })
      return crumbs
    }
    if (head === "roles") {
      crumbs.push({ label: t("nav.roles"), to: "/roles" })
      if (parts[1]) crumbs.push({ label: t("roles.detail") })
      return crumbs
    }
    crumbs.push({ label: t("nav.permissions"), to: "/permissions" })
    return crumbs
  }
  if (system.has(head)) {
    crumbs.push({ label: t("nav.system") })
    if (head === "mail") {
      if (parts[1] === "campaigns") {
        crumbs.push({ label: t("nav.mailCampaigns"), to: "/mail/campaigns" })
        if (parts[2] === "new") crumbs.push({ label: t("mail.createCampaign") })
        else if (parts[2]) crumbs.push({ label: t("mail.editCampaign") })
        return crumbs
      }
      crumbs.push({ label: t("nav.mailJobs"), to: "/mail/jobs" })
      return crumbs
    }
    const label =
      head === "dicts" ? t("nav.dicts") : head === "configs" ? t("nav.configs") : t("nav.logs")
    crumbs.push({ label, to: `/${head}` })
    return crumbs
  }

  if (head === "settings") {
    crumbs.push({ label: t("nav.settings"), to: "/settings" })
    if (parts[1] === "password") crumbs.push({ label: t("nav.password"), to: "/settings/password" })
    return crumbs
  }
  return crumbs
}
