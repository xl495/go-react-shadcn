import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useI18n } from "@/lib/i18n"
import type { AuthSettings } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"
import { GuestChrome } from "@/components/GuestChrome"

export function ForgotPasswordPage() {
  const { t } = useI18n()
  const [email, setEmail] = useState("")
  const [error, setError] = useState("")
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(false)
  const [settings, setSettings] = useState<AuthSettings | undefined>()
  const [settingsReady, setSettingsReady] = useState(false)
  const challengeRef = useRef<AuthChallengeHandle>(null)

  useEffect(() => {
    api.settings().then(setSettings).catch(() => undefined).finally(() => setSettingsReady(true))
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const challenge = (await challengeRef.current?.collect()) ?? {}
      await api.forgotPassword({ email, ...challenge })
      setDone(true)
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      setError(err instanceof ApiError ? err.message || t("forgot.failed") : t("forgot.failed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestChrome>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("forgot.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("forgot.subtitle")}</p>
        </div>
        {done ? (
          <p className="text-sm text-muted-foreground">{t("forgot.sent")}</p>
        ) : (
          <>
            <div className="grid gap-2">
              <label htmlFor="email" className="text-sm font-medium">
                {t("forgot.email")}
              </label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
            <AuthChallenge ref={challengeRef} settings={settingsReady ? (settings ?? { captchaProvider: "image" }) : undefined} action="forgot" />
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <button
              type="submit"
              disabled={loading || !settingsReady}
              className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
            >
              {loading ? t("forgot.submitting") : t("forgot.submit")}
            </button>
          </>
        )}
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("forgot.back")}
        </Link>
      </form>
    </GuestChrome>
  )
}

export function ResetPasswordPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    if (!token) {
      setError(t("reset.invalid"))
      return
    }
    setLoading(true)
    try {
      await api.resetPassword({ token, newPassword: password })
      navigate("/login", { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("reset.failed") : t("reset.failed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestChrome>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("reset.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("reset.subtitle")}</p>
        </div>
        {!token ? <p className="text-sm text-destructive">{t("reset.invalid")}</p> : null}
        <div className="grid gap-2">
          <label htmlFor="np" className="text-sm font-medium">
            {t("reset.password")}
          </label>
          <input
            id="np"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <button
          type="submit"
          disabled={loading || !token}
          className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
        >
          {loading ? t("reset.submitting") : t("reset.submit")}
        </button>
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("reset.back")}
        </Link>
      </form>
    </GuestChrome>
  )
}
