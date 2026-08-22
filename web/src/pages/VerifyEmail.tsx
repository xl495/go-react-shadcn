import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"
import { GuestChrome } from "@/components/GuestChrome"

export function VerifyEmailPage() {
  const { t } = useI18n()
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(Boolean(token))

  useEffect(() => {
    if (!token) return
    let cancelled = false
    api
      .verifyEmail({ token })
      .then((result) => {
        if (cancelled) return
        if (!result.token || !result.user) {
          setError(t("verify.failed"))
          return
        }
        login(result.token, result.user)
        navigate("/", { replace: true })
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message || t("verify.failed") : t("verify.failed"))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [token, login, navigate, t])

  return (
    <GuestChrome>
      <div className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("verify.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("verify.subtitle")}</p>
        </div>
        {loading ? <p className="text-sm text-muted-foreground">{t("verify.submitting")}</p> : null}
        {!token ? <p className="text-sm text-destructive">{t("verify.invalid")}</p> : null}
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("verify.back")}
        </Link>
      </div>
    </GuestChrome>
  )
}
