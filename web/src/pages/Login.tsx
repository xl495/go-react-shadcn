import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"
import type { AuthSettings, LoginResult, User } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"
import { GoogleSignIn } from "@/components/GoogleSignIn"
import { GuestChrome } from "@/components/GuestChrome"
import { safeInternalPath } from "@/lib/safe-path"

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
  const [totp, setTotp] = useState<{ ticket: string; enroll: boolean } | null>(null)
  const [code, setCode] = useState("")
  const [secret, setSecret] = useState("")
  const [otpauth, setOtpauth] = useState("")

  useEffect(() => {
    api.settings().then(setSettings).catch(() => undefined).finally(() => setSettingsReady(true))
  }, [])

  function enter(token: string, user: User) {
    login(token, user)
    navigate(from, { replace: true })
  }

  async function applyResult(result: LoginResult) {
    if (result.totpRequired && result.totpTicket) {
      setTotp({ ticket: result.totpTicket, enroll: !!result.totpEnroll })
      if (result.totpEnroll) {
        const setup = await api.totpSetup({ ticket: result.totpTicket })
        setSecret(setup.secret)
        setOtpauth(setup.otpauthUri)
      }
      return
    }
    if (!result.token || !result.user) {
      setError(t("login.failed"))
      return
    }
    enter(result.token, result.user)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const challenge = (await challengeRef.current?.collect()) ?? {}
      await applyResult(await api.login({ username, password, ...challenge }))
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
      await applyResult(await api.google({ idToken, client: "web" }))
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("login.failed") : t("login.failed"))
    } finally {
      setLoading(false)
    }
  }

  async function onTotp(e: FormEvent) {
    e.preventDefault()
    if (!totp) return
    setError("")
    setLoading(true)
    try {
      const result = totp.enroll
        ? await api.totpConfirm({ ticket: totp.ticket, code })
        : await api.totpVerify({ ticket: totp.ticket, code })
      await applyResult(result)
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("login.failed") : t("login.failed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestChrome>
      {totp ? (
        <form onSubmit={onTotp} className="w-full max-w-sm space-y-5">
          <div>
            <p className="font-display text-[13px] text-muted-foreground">gra</p>
            <h1 className="mt-1 font-display text-[2rem] leading-none tracking-tight">{t("login.totpTitle")}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{totp.enroll ? t("login.totpEnroll") : t("login.totpHint")}</p>
          </div>
          {totp.enroll && secret ? (
            <div className="space-y-2 text-sm">
              <p className="break-all font-mono text-xs">{otpauth}</p>
              <p>
                {t("login.totpSecret")}: <span className="font-mono">{secret}</span>
              </p>
            </div>
          ) : null}
          <input
            inputMode="numeric"
            autoComplete="one-time-code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="h-10 w-full rounded-md border bg-background px-3 text-sm"
          />
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <button type="submit" disabled={loading} className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60">
            {loading ? t("login.submitting") : t("login.totpContinue")}
          </button>
        </form>
      ) : (
        <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <p className="font-display text-[13px] text-muted-foreground">gra</p>
          <h1 className="mt-1 font-display text-[2rem] leading-none tracking-tight">{t("login.title")}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t("login.subtitle")}</p>
          {settings?.maintenance ? (
            <p className="mt-2 text-sm text-destructive">{t("login.maintenance")}</p>
          ) : null}
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
      )}
    </GuestChrome>
  )
}
