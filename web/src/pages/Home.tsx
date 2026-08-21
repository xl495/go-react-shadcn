import { useI18n } from "@/lib/i18n"
import { useAuth } from "@/lib/auth"

export function HomePage() {
  const { t } = useI18n()
  const { user } = useAuth()
  const displayName = user?.nickname || user?.username || "—"
  const initial = displayName.slice(0, 1).toUpperCase()

  return (
    <section className="rounded-lg border p-6">
      <div className="flex items-center gap-4">
        {user?.avatar ? (
          <img src={user.avatar} alt={displayName} className="size-16 rounded-full border object-cover" />
        ) : (
          <div className="flex size-16 items-center justify-center rounded-full border bg-muted text-xl font-semibold">
            {initial}
          </div>
        )}
        <div className="min-w-0">
          <h1 className="truncate text-xl font-semibold tracking-tight">{displayName}</h1>
          {user?.nickname && user.username ? (
            <p className="truncate text-sm text-muted-foreground">@{user.username}</p>
          ) : null}
        </div>
      </div>
      <dl className="mt-6 grid gap-3 text-sm">
        <div className="flex justify-between gap-4 border-t pt-3">
          <dt className="text-muted-foreground">{t("home.username")}</dt>
          <dd>{user?.username || "—"}</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">{t("home.phone")}</dt>
          <dd>{user?.phone || t("home.unbound")}</dd>
        </div>
      </dl>
    </section>
  )
}
