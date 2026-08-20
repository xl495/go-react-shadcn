import { useState, type FormEvent, type ReactNode } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { ApiError } from "@/api/client"
import { api } from "@/api/client"
import { translateApiError, useI18n } from "@/providers/i18n"
import { Button } from "@/components/ui/button"

function GuestShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-full flex-col overflow-y-auto bg-background">
      <header className="flex h-14 items-center justify-between border-b px-6">
        <span className="text-sm font-semibold">Latch</span>
        <LanguageSwitcher />
      </header>
      <div className="flex flex-1 items-center justify-center px-4">
        <div className="w-full max-w-sm space-y-5">{children}</div>
      </div>
    </div>
  )
}

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
    <GuestShell>
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
    </GuestShell>
  )
}
