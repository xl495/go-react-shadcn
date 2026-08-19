import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { DICT, useDict } from "@/lib/dict"
import { formatDateTime } from "@/lib/format"
import { roleLabel, translateApiError, useI18n } from "@/lib/i18n"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { DashboardStats } from "@/lib/types"

export function DashboardPage() {
  const { user } = useAuth()
  const { t } = useI18n()
  const deptDict = useDict(DICT.department)
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    api.stats().then(setStats).catch((e) => setError(translateApiError(e, t)))
  }, [t])

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("dashboard.title")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("dashboard.hello", { name: user?.nickname || user?.username || "" })}
        </p>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat title={t("dashboard.users")} value={stats?.users} hint={t("dashboard.usersHint")} />
        <Stat title={t("dashboard.roles")} value={stats?.roles} hint={t("dashboard.rolesHint")} />
        <Stat title={t("dashboard.permissions")} value={stats?.permissions} hint={t("dashboard.permsHint")} />
        <Stat title={t("nav.dicts")} value={stats?.dicts} hint={t("dashboard.dictsHint")} />
        <Stat title={t("nav.configs")} value={stats?.configs} hint={t("dashboard.configsHint")} />
        <Stat title={t("nav.logs")} value={stats?.logs} hint={t("dashboard.logsHint")} />
      </div>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("dashboard.session")}</CardTitle>
          <CardDescription>{t("dashboard.sessionHint")}</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
          <Field label={t("users.account")} value={user?.username} />
          <Field label={t("users.nickname")} value={user?.nickname} />
          <Field
            label={t("users.roles")}
            value={(user?.roles ?? []).map((r) => roleLabel(r.code, r.name, t)).join(" · ")}
          />
          <Field label={t("users.lastLogin")} value={formatDateTime(user?.lastLoginAt)} />
          <Field label={t("users.email")} value={user?.email} />
          <Field label={t("users.phone")} value={user?.phone} />
          <Field label={t("users.department")} value={deptDict.label(user?.department)} />
          <Field label={t("users.jobTitle")} value={user?.title} />
        </CardContent>
      </Card>
    </div>
  )
}

function Stat({ title, value, hint }: { title: string; value?: number; hint: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{hint}</CardDescription>
        <CardTitle className="text-3xl tabular-nums">{value ?? "—"}</CardTitle>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">{title}</CardContent>
    </Card>
  )
}

function Field({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 font-medium">{value || "—"}</p>
    </div>
  )
}
