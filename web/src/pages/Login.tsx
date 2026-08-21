import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"
import type { AuthSettings } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"
import { GoogleSignIn } from "@/components/GoogleSignIn"
import { GuestChrome } from "@/components/GuestChrome"
import { safeInternalPath } from "@latch/auth/safe-path"

export function LoginPage() {
  const { login } = useAuth()
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const from = safeInternalPath((location.state as { from?: string } | null)?.from)

  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
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
      const result = await api.login({ username, password, ...challenge })
      login(result.token, result.user)
      navigate(from, { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      setError(err instanceof ApiError ? err.message || t("login.failed") : t("login.failed"))
    } finally {
      setLoading(false)
    }
  }

  async function onGoogle(idToken: string) {
    setError("")
    setLoading(true)
    try {
      const result = await api.google({ idToken, client: "web" })
      login(result.token, result.user)
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("login.failed") : t("login.failed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestChrome>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <p className="font-display text-[13px] text-muted-foreground">gra</p>
          <h1 className="mt-1 font-display text-[2rem] leading-none tracking-tight">{t("login.title")}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t("login.subtitle")}</p>
        </div>
        {settings?.googleEnabled ? (
          <div className="space-y-3">
            <GoogleSignIn
              clientId={settings.googleClientId}
              locale={locale}
              onCredential={onGoogle}
              disabled={loading || !settingsReady}
            />
            {settings.googleRegisterEnabled ? (
              <p className="text-center text-xs text-muted-foreground">{t("login.googleRegister")}</p>
            ) : null}
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              <span className="h-px flex-1 bg-border" />
              {t("login.or")}
              <span className="h-px flex-1 bg-border" />
            </div>
          </div>
        ) : null}
        <div className="grid gap-2">
          <label htmlFor="username" className="text-sm font-medium">
            {t("login.username")}
          </label>
          <input
            id="username"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
        <div className="grid gap-2">
          <label htmlFor="password" className="text-sm font-medium">
            {t("login.password")}
          </label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <Link to="/forgot-password" className="text-xs text-primary underline-offset-4 hover:underline">
            {t("login.forgot")}
          </Link>
        </div>
        <AuthChallenge ref={challengeRef} settings={settingsReady ? (settings ?? { captchaProvider: "image" }) : undefined} action="login" />
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <button
          type="submit"
          disabled={loading || !settingsReady}
          className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
        >
          {loading ? t("login.submitting") : t("login.submit")}
        </button>
        {settings?.registerEnabled !== false || settings?.googleRegisterEnabled ? (
          <p className="text-center text-sm text-muted-foreground">
            {t("login.noAccount")}{" "}
            <Link to="/register" className="text-foreground underline-offset-4 hover:underline">
              {t("login.register")}
            </Link>
          </p>
        ) : null}
      </form>
    </GuestChrome>
  )
}
