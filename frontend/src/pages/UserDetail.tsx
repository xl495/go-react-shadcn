import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { api } from "@/lib/api"
import { DICT, useDict } from "@/lib/dict"
import { formatDateTime } from "@/lib/format"
import { roleLabel, translateApiError, useI18n } from "@/lib/i18n"
import { Avatar } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import type { User } from "@/lib/types"

export function UserDetailPage() {
  const { id } = useParams()
  const { t } = useI18n()
  const genderDict = useDict(DICT.gender)
  const statusDict = useDict(DICT.userStatus)
  const deptDict = useDict(DICT.department)
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    const n = Number(id)
    if (!n) {
      setError(t("errors.40410"))
      return
    }
    api
      .getUser(n)
      .then(setUser)
      .catch((e: Error) => setError(translateApiError(e, t)))
  }, [id, t])

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold tracking-tight">{t("users.detail")}</h2>
        <Button asChild variant="outline" size="sm">
          <Link to="/users">{t("users.backToList")}</Link>
        </Button>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {user ? (
        <Card>
          <CardHeader className="flex flex-row items-center gap-4">
            <Avatar name={user.nickname || user.username} src={user.avatar} className="size-16 text-lg" />
            <div>
              <CardTitle>{user.nickname || user.username}</CardTitle>
              <p className="mt-1 text-sm text-muted-foreground">{user.username}</p>
            </div>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm sm:grid-cols-2">
            <Field label={t("users.phone")} value={user.phone} />
            <Field label={t("users.email")} value={user.email} />
            <Field label={t("users.gender")} value={genderDict.label(user.gender)} />
            <Field label={t("users.department")} value={deptDict.label(user.department)} />
            <Field label={t("users.jobTitle")} value={user.title} />
            <Field label={t("app.status")} value={statusDict.label(user.status)} />
            <Field
              label={t("users.roles")}
              value={(user.roles ?? []).map((r) => roleLabel(r.code, r.name, t)).join(" · ")}
            />
            <Field label={t("users.lastLogin")} value={formatDateTime(user.lastLoginAt)} />
            <Field label="IP" value={user.lastLoginIp} />
            <Field label={t("users.createdAt")} value={formatDateTime(user.createdAt)} />
            <div className="sm:col-span-2">
              <Field label={t("users.remark")} value={user.remark} />
            </div>
            {user.avatar ? (
              <div className="sm:col-span-2">
                <p className="text-xs text-muted-foreground">{t("users.avatar")}</p>
                <p className="mt-0.5 font-mono text-xs break-all">{user.avatar}</p>
              </div>
            ) : null}
            <div className="sm:col-span-2">
              <Badge variant="muted">ID {user.id}</Badge>
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function Field({ label, value }: { label: string; value?: string | null }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 font-medium">{value || "—"}</p>
    </div>
  )
}
