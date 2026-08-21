import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { GuestChrome } from "@/components/layout/GuestChrome"
import { ApiError } from "@/api/client"
import { api } from "@/api/client"
import { translateApiError, useI18n } from "@/providers/i18n"
import { Button } from "@/components/ui/button"

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
      setError(err instanceof ApiError ? translateApiError(err, t) : t("mail.unsubFailed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <GuestChrome>
      {done ? (
        <div className="space-y-4">
          <h1 className="text-2xl font-semibold tracking-tight">{t("mail.unsubTitle")}</h1>
          <p className="text-sm text-muted-foreground">{t("mail.unsubDone")}</p>
          <Button variant="link" className="h-auto px-0" asChild>
            <Link to="/login">{t("login.backToLogin")}</Link>
          </Button>
        </div>
      ) : (
        <form onSubmit={onSubmit} className="space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("mail.unsubTitle")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("mail.unsubSubtitle")}</p>
          </div>
          {!token ? <p className="text-sm text-destructive">{t("mail.unsubMissing")}</p> : null}
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <Button type="submit" disabled={pending || !token} className="h-10 w-full">
            {pending ? t("mail.unsubSubmitting") : t("mail.unsubSubmit")}
          </Button>
        </form>
      )}
    </GuestChrome>
  )
}
