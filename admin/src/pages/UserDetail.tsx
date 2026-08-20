import { Link, useParams } from "react-router-dom"
import { DICT, useDict } from "@/hooks/dict"
import { formatDateTime } from "@/utils/format"
import { roleLabel, translateApiError, useI18n } from "@/providers/i18n"
import { useUser } from "@/hooks/queries"
import { Avatar } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { PageFallback } from "@/components/PageFallback"

export function UserDetailPage() {
  const { id } = useParams()
  const { t } = useI18n()
  const genderDict = useDict(DICT.gender)
  const statusDict = useDict(DICT.userStatus)
  const deptDict = useDict(DICT.department)
  const userId = Number(id)
  const { data: user, error, isLoading } = useUser(userId)

  if (!userId) {
    return <p className="text-sm text-destructive">{t("errors.40410")}</p>
  }
  if (isLoading) return <PageFallback />
  if (error) return <p className="text-sm text-destructive">{translateApiError(error as Error, t)}</p>
  if (!user) return null

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold tracking-tight">{t("users.detail")}</h2>
        <Button asChild variant="outline" size="sm">
          <Link to={user.kind === "web" ? "/web-users" : "/users"}>{t("users.backToList")}</Link>
        </Button>
      </div>
      <Card>
        <CardHeader className="flex flex-row items-center gap-4">
          <Avatar src={user.avatar} name={user.nickname || user.username} className="size-14" />
          <div>
            <CardTitle>{user.nickname || user.username}</CardTitle>
            <p className="text-sm text-muted-foreground">@{user.username}</p>
          </div>
          <Badge className="ml-auto">{statusDict.label(user.status)}</Badge>
        </CardHeader>
      </Card>
      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("users.account")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <Field label={t("login.username")} value={user.username} />
            <Field label={t("users.nickname")} value={user.nickname} />
            <Field
              label={t("users.roles")}
              value={(user.roles ?? []).map((r) => roleLabel(r.code, r.name, t)).join(" · ")}
            />
            <Field label={t("app.status")} value={statusDict.label(user.status)} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("users.contact")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <Field label={t("users.email")} value={user.email} />
            <Field label={t("users.phone")} value={user.phone} />
            <Field label={t("users.gender")} value={genderDict.label(user.gender)} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("users.orgLogin")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <Field label={t("users.department")} value={deptDict.label(user.department)} />
            <Field label={t("users.jobTitle")} value={user.title} />
            <Field label={t("users.timezone")} value={user.timezone} />
            <Field
              label={t("users.lastLogin")}
              value={
                [formatDateTime(user.lastLoginAt), user.lastLoginIp].filter(Boolean).join(" · ") || undefined
              }
            />
            <Field
              label={t("users.marketingOptIn")}
              value={user.marketingOptIn ? t("app.active") : t("app.disabled")}
            />
            <Field label={t("users.remark")} value={user.remark} />
          </CardContent>
        </Card>
      </div>
    </div>
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
