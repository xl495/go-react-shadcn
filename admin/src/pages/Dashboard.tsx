import { useAuth } from "@/providers/auth"
import { DICT, useDict } from "@/hooks/dict"
import { formatDateTime } from "@/utils/format"
import { roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { useDashboardStats } from "@/hooks/queries"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { DashboardStats, LoginLog } from "@/types"

type OpsStats = DashboardStats & {
  mailQueued?: number
  recentLogins?: LoginLog[]
  failedLogins?: LoginLog[]
}

export function DashboardPage() {
  const { user } = useAuth()
  const { t } = useI18n()
  const deptDict = useDict(DICT.department)
  const { data, error, isLoading } = useDashboardStats()
  const stats = data as OpsStats | undefined

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("dashboard.title")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("dashboard.hello", { name: user?.nickname || user?.username || "" })}
        </p>
      </div>
      {error ? (
        <p className="text-sm text-destructive">{translateApiError(error as Error, t)}</p>
      ) : null}
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat title={t("dashboard.users")} value={stats?.users} hint={t("dashboard.usersHint")} loading={isLoading} />
        <Stat title={t("dashboard.roles")} value={stats?.roles} hint={t("dashboard.rolesHint")} loading={isLoading} />
        <Stat title={t("dashboard.permissions")} value={stats?.permissions} hint={t("dashboard.permsHint")} loading={isLoading} />
        <Stat title={t("nav.dicts")} value={stats?.dicts} hint={t("dashboard.dictsHint")} loading={isLoading} />
        <Stat title={t("nav.configs")} value={stats?.configs} hint={t("dashboard.configsHint")} loading={isLoading} />
        <Stat title={t("nav.logs")} value={stats?.logs} hint={t("dashboard.logsHint")} loading={isLoading} />
        <Stat title={t("dashboard.mailQueued")} value={stats?.mailQueued} hint={t("dashboard.mailQueuedHint")} loading={isLoading} />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <EventList title={t("dashboard.recentLogins")} rows={stats?.recentLogins} loading={isLoading} />
        <EventList title={t("dashboard.failedLogins")} rows={stats?.failedLogins} loading={isLoading} />
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

function Stat({ title, value, hint, loading }: { title: string; value?: number; hint: string; loading?: boolean }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{hint}</CardDescription>
        <CardTitle className="text-3xl tabular-nums">
          {loading ? <span className="inline-block h-8 w-16 animate-pulse rounded-md bg-muted" /> : (value ?? "—")}
        </CardTitle>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">{title}</CardContent>
    </Card>
  )
}

function EventList({ title, rows, loading }: { title: string; rows?: LoginLog[]; loading?: boolean }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        {loading ? (
          <div className="space-y-2">
            <div className="h-5 animate-pulse rounded-md bg-muted" />
            <div className="h-5 animate-pulse rounded-md bg-muted" />
            <div className="h-5 w-2/3 animate-pulse rounded-md bg-muted" />
          </div>
        ) : (rows ?? []).length === 0 ? (
          <p className="text-muted-foreground">—</p>
        ) : (
          (rows ?? []).map((row) => (
            <div key={row.id} className="flex justify-between gap-3 border-b pb-2 last:border-0">
              <span className="truncate font-medium">{row.username}</span>
              <span className="shrink-0 text-xs text-muted-foreground">
                {row.status} · {row.ip || "—"} · {formatDateTime(row.createdAt)}
              </span>
            </div>
          ))
        )}
      </CardContent>
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
