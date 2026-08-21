import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useI18n } from "@/lib/i18n"
import { GuestChrome } from "@/components/GuestChrome"

export function UnsubscribePage() {
  const { t } = useI18n()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [pending, setPending] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState("")

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setPending(true)
    try {
      await api.unsubscribe(token)
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("unsub.failed") : t("unsub.failed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <GuestChrome
      trailing={
        <Link to="/login" className="text-sm text-muted-foreground hover:text-foreground">
          {t("unsub.login")}
        </Link>
      }
    >
      <div className="w-full max-w-sm">
        {done ? (
          <section className="space-y-3">
            <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("unsub.title")}</h1>
            <p className="text-sm text-muted-foreground">{t("unsub.done")}</p>
          </section>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <div>
              <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("unsub.title")}</h1>
              <p className="mt-1 text-sm text-muted-foreground">{t("unsub.confirm")}</p>
            </div>
            {!token ? <p className="text-sm text-destructive">{t("unsub.invalid")}</p> : null}
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <button
              type="submit"
              disabled={pending || !token}
              className="h-10 w-full rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
            >
              {pending ? t("unsub.submitting") : t("unsub.submit")}
            </button>
          </form>
        )}
      </div>
    </GuestChrome>
  )
}
