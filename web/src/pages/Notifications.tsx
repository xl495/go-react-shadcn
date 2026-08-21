import { useCallback, useEffect, useState } from "react"
import { api, type NotificationItem } from "@/lib/api"
import { useI18n } from "@/lib/i18n"

export function NotificationsPage() {
  const { t } = useI18n()
  const [items, setItems] = useState<NotificationItem[]>([])
  const [error, setError] = useState("")

  const load = useCallback(() => {
    api
      .notifications()
      .then((page) => setItems(page.items ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : t("app.loadFailed")))
  }, [t])

  useEffect(() => {
    load()
    const timer = window.setInterval(load, 30_000)
    return () => window.clearInterval(timer)
  }, [load])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">{t("notify.title")}</h1>
        <button
          type="button"
          className="rounded-md border px-3 py-1.5 text-sm hover:bg-muted"
          onClick={() => void api.readAllNotifications().then(load)}
        >
          {t("notify.readAll")}
        </button>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("notify.empty")}</p>
      ) : (
        <ul className="space-y-3">
          {items.map((row) => (
            <li key={row.id} className={`rounded-md border p-3 ${row.readAt ? "text-muted-foreground" : ""}`}>
              <p className="font-medium">{row.title}</p>
              {row.body ? <p className="mt-1 text-sm">{row.body}</p> : null}
              {row.readAt ? null : (
                <button
                  type="button"
                  className="mt-2 text-xs underline"
                  onClick={() => void api.readNotification(row.id).then(load)}
                >
                  {t("notify.markRead")}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
