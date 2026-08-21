import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"
import type { AuthSettings } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"
import { GoogleSignIn } from "@/components/GoogleSignIn"
import { GuestChrome } from "@/components/GuestChrome"

export function RegisterPage() {
  const { login } = useAuth()
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const [settings, setSettings] = useState<AuthSettings | undefined>()
  const [settingsReady, setSettingsReady] = useState(false)
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [pendingEmail, setPendingEmail] = useState("")
  const [loading, setLoading] = useState(false)
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
      const result = await api.register({ username, email, password, ...challenge })
      if ("pending" in result && result.pending) {
        setPendingEmail(result.email || email)
        return
      }
      if ("token" in result) {
        login(result.token, result.user)
        navigate("/", { replace: true })
      }
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      setError(err instanceof ApiError ? err.message || t("register.failed") : t("register.failed"))
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
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("register.failed") : t("register.failed"))
    } finally {
      setLoading(false)
    }
  }

  const passwordOn = settings?.registerEnabled !== false
  const googleOn = Boolean(settings?.googleEnabled && settings.googleRegisterEnabled)

  return (
    <GuestChrome>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("register.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("register.subtitle")}</p>
        </div>
        {pendingEmail ? (
          <p className="text-sm text-muted-foreground">{t("register.checkEmail")}</p>
        ) : null}
        {googleOn ? (
          <div className="space-y-3">
            <GoogleSignIn
              clientId={settings?.googleClientId ?? ""}
              locale={locale}
              onCredential={onGoogle}
              disabled={loading}
            />
            {passwordOn ? (
              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span className="h-px flex-1 bg-border" />
                {t("login.or")}
                <span className="h-px flex-1 bg-border" />
              </div>
            ) : null}
          </div>
        ) : null}
        {passwordOn ? (
          <>
            <div className="grid gap-2">
              <label htmlFor="username" className="text-sm font-medium">
                {t("login.username")}
              </label>
              <input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
            <div className="grid gap-2">
              <label htmlFor="email" className="text-sm font-medium">
                {t("register.email")}
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
            <div className="grid gap-2">
              <label htmlFor="password" className="text-sm font-medium">
                {t("register.password")}
              </label>
              <input
                id="password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
            <AuthChallenge ref={challengeRef} settings={settingsReady ? (settings ?? { captchaProvider: "image" }) : undefined} action="register" />
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <button
              type="submit"
              disabled={loading || !settingsReady}
              className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
            >
              {loading ? t("register.submitting") : t("register.submit")}
            </button>
          </>
        ) : error ? (
          <p className="text-sm text-destructive">{error}</p>
        ) : null}
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("register.back")}
        </Link>
      </form>
    </GuestChrome>
  )
}
