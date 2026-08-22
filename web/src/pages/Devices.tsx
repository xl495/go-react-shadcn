import { useEffect, useState } from "react"
import { api, ApiError } from "@/lib/api"
import { useI18n } from "@/lib/i18n"

type Session = {
  id: number
  ip: string
  userAgent: string
  createdAt: string
  revokedAt?: string | null
  current?: boolean
}
type LoginRow = { id: number; status: string; ip: string; createdAt: string }

export function DevicesPage() {
  const { t } = useI18n()
  const [sessions, setSessions] = useState<Session[]>([])
  const [logs, setLogs] = useState<LoginRow[]>([])
  const [error, setError] = useState("")

  useEffect(() => {
    let cancelled = false
    Promise.all([api.ownSessions(), api.ownLoginLogs()])
      .then(([nextSessions, nextLogs]) => {
        if (cancelled) return
        setSessions(nextSessions)
        setLogs(nextLogs.items)
        setError("")
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof ApiError ? err.message : t("app.loadFailed"))
      })
    return () => {
      cancelled = true
    }
  }, [t])

  return (
    <div className="grid gap-6">
      <section className="grid gap-3 rounded-lg border p-6">
        <h1 className="text-xl font-semibold tracking-tight">{t("devices.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("devices.subtitle")}</p>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        {sessions.map((row) => (
          <div key={row.id} className="flex items-center justify-between gap-3 border-b py-2 last:border-0">
            <div className="min-w-0">
              <p className="font-mono text-xs">
                {row.ip || "—"}
                {row.current ? ` · ${t("devices.current")}` : ""}
              </p>
              <p className="truncate text-xs text-muted-foreground">{row.userAgent}</p>
            </div>
            {row.revokedAt ? (
              <span className="text-xs text-muted-foreground">{t("devices.revoked")}</span>
            ) : row.current ? (
              <span className="text-xs text-muted-foreground">{t("devices.current")}</span>
            ) : (
              <button
                type="button"
                className="text-sm text-primary underline-offset-4 hover:underline"
                onClick={() =>
                  api
                    .revokeOwnSession(row.id)
                    .then(() => api.ownSessions().then(setSessions))
                    .catch((err) => {
                      setError(err instanceof ApiError ? err.message : t("app.loadFailed"))
                    })
                }
              >
                {t("devices.kick")}
              </button>
            )}
          </div>
        ))}
      </section>
      <section className="grid gap-3 rounded-lg border p-6">
        <h2 className="text-lg font-semibold">{t("devices.logs")}</h2>
        {logs.map((row) => (
          <div key={row.id} className="flex justify-between gap-3 text-sm">
            <span>{row.status}</span>
            <span className="font-mono text-xs text-muted-foreground">{row.ip}</span>
            <span className="text-xs text-muted-foreground">{row.createdAt}</span>
          </div>
        ))}
      </section>
    </div>
  )
}
