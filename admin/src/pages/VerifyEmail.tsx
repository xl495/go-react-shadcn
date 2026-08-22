import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { GuestChrome } from "@/components/layout/GuestChrome"
import { api, ApiError, getToken } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { Button } from "@/components/ui/button"

export function VerifyEmailPage() {
  const { t } = useI18n()
  const { login, updateUser } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [error, setError] = useState("")
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(Boolean(token))
  const signedIn = Boolean(getToken())

  useEffect(() => {
    if (!token) return
    let cancelled = false
    api
      .verifyEmail(token)
      .then((result) => {
        if (cancelled) return
        if (result.changed) {
          if (result.user && getToken()) updateUser(result.user)
          setDone(true)
          return
        }
        if (result.token && result.user) {
          login(result.token, result.user)
          navigate("/", { replace: true })
          return
        }
        setError(t("login.verifyFailed"))
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message || t("login.verifyFailed") : t("login.verifyFailed"))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [token, login, navigate, t, updateUser])

  return (
    <GuestChrome>
      <div className="space-y-5">
        <div>
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("login.verifyTitle")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("login.verifySubtitle")}</p>
        </div>
        {loading ? <p className="text-sm text-muted-foreground">{t("login.verifySubmitting")}</p> : null}
        {!token ? <p className="text-sm text-destructive">{t("login.verifyInvalid")}</p> : null}
        {done ? <p className="text-sm text-muted-foreground">{t("login.verifyChanged")}</p> : null}
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <Button variant="link" className="h-auto px-0" asChild>
          <Link to={signedIn ? "/settings" : "/login"}>{signedIn ? t("login.verifySettings") : t("login.backToLogin")}</Link>
        </Button>
      </div>
    </GuestChrome>
  )
}
