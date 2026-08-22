import { useRef, useState, type FormEvent } from "react"
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { GuestChrome } from "@/components/layout/GuestChrome"
import { api, ApiError } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { useAuthSettings, useGoogleAuthMutation, useLoginMutation } from "@/hooks/queries"
import { PageFallback } from "@/components/PageFallback"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/auth/AuthChallenge"
import { GoogleSignIn } from "@/components/auth/GoogleSignIn"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { safeInternalPath } from "@/lib/safe-path"
import type { LoginResult, User } from "@/types"

export function LoginPage() {
  const { user, loading, login } = useAuth()
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const fromRaw = (location.state as { from?: string } | null)?.from
  const from = safeInternalPath(fromRaw)

  const [username, setUsername] = useState(import.meta.env.DEV ? "admin" : "")
  const [password, setPassword] = useState(import.meta.env.DEV ? "admin123" : "")
  const settings = useAuthSettings()
  const loginMut = useLoginMutation()
  const googleMut = useGoogleAuthMutation()
  const challengeRef = useRef<AuthChallengeHandle>(null)
  const settingsReady = !settings.isPending
  const [totp, setTotp] = useState<{ ticket: string; enroll: boolean } | null>(null)
  const [code, setCode] = useState("")
  const [secret, setSecret] = useState("")
  const [otpauth, setOtpauth] = useState("")
  const [recovery, setRecovery] = useState<string[] | null>(null)
  const [pendingLogin, setPendingLogin] = useState<{ token: string; user: User } | null>(null)
  const [totpBusy, setTotpBusy] = useState(false)
  const pending = loginMut.isPending || googleMut.isPending || totpBusy

  if (loading) return <PageFallback />
  if (user) return <Navigate to="/" replace />

  function enter(token: string, nextUser: User) {
    login(token, nextUser)
    toast.success(t("login.submit"))
    navigate(from, { replace: true })
  }

  async function applyResult(result: LoginResult) {
    if (result.totpRequired && result.totpTicket) {
      setTotp({ ticket: result.totpTicket, enroll: !!result.totpEnroll })
      if (result.totpEnroll) {
        const setup = await api.setup({ ticket: result.totpTicket })
        setSecret(setup.secret)
        setOtpauth(setup.otpauthUri)
      }
      return
    }
    if (!result.token || !result.user) {
      toast.error(t("login.failed"))
      return
    }
    if (result.recoveryCodes?.length) {
      setPendingLogin({ token: result.token, user: result.user })
      setRecovery(result.recoveryCodes)
      return
    }
    enter(result.token, result.user)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    try {
      const challenge = (await challengeRef.current?.collect()) ?? {}
      const result = await loginMut.mutateAsync({
        username,
        password,
        client: "admin",
        ...challenge,
      })
      await applyResult(result)
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
        return
      }
      if (!(err instanceof ApiError)) toast.error(t("login.failed"))
    }
  }

  async function onGoogle(idToken: string) {
    try {
      const result = await googleMut.mutateAsync({ idToken, client: "admin" })
      await applyResult(result)
    } catch (err) {
      if (!(err instanceof ApiError)) toast.error(t("login.failed"))
    }
  }

  async function onTotp(e: FormEvent) {
    e.preventDefault()
    if (!totp) return
    setTotpBusy(true)
    try {
      if (totp.enroll) {
        const result = (await api.confirm({ ticket: totp.ticket, code })) as LoginResult
        await applyResult(result)
      } else {
        const result = (await api.verify({ ticket: totp.ticket, code })) as LoginResult
        await applyResult(result)
      }
    } catch (err) {
      if (!(err instanceof ApiError)) toast.error(t("login.failed"))
    } finally {
      setTotpBusy(false)
    }
  }

  return (
    <GuestChrome>
      {recovery && pendingLogin ? (
        <div className="space-y-5">
          <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("login.totpTitle")}</h1>
          <p className="text-sm text-muted-foreground">{t("login.totpRecovery")}</p>
          <ul className="grid gap-1 font-mono text-sm">
            {recovery.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
          <Button
            type="button"
            className="h-10 w-full"
            onClick={() => enter(pendingLogin.token, pendingLogin.user)}
          >
            {t("login.totpContinue")}
          </Button>
        </div>
      ) : totp ? (
        <form onSubmit={onTotp} className="space-y-5">
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
          <div className="grid gap-2">
            <Label htmlFor="totp">{t("login.totpTitle")}</Label>
            <Input id="totp" inputMode="numeric" autoComplete="one-time-code" value={code} onChange={(e) => setCode(e.target.value)} />
          </div>
          <Button type="submit" disabled={pending || code.trim().length < 6} className="h-10 w-full">
            {pending ? t("login.submitting") : t("login.totpContinue")}
          </Button>
        </form>
      ) : (
        <form onSubmit={onSubmit} className="space-y-5">
          <div>
            <p className="font-display text-[13px] text-muted-foreground">gra</p>
            <h1 className="mt-1 font-display text-[2rem] leading-none tracking-tight">{t("login.title")}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t("login.subtitle")}</p>
            {settings.data?.maintenance ? (
              <p className="mt-2 text-sm text-destructive">{t("login.maintenance")}</p>
            ) : null}
          </div>
          {settings.data?.googleEnabled ? (
            <div className="space-y-3">
              <GoogleSignIn clientId={settings.data.googleClientId} locale={locale} onCredential={onGoogle} disabled={pending || !settingsReady} />
              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span className="h-px flex-1 bg-border" />
                {t("login.or")}
                <span className="h-px flex-1 bg-border" />
              </div>
            </div>
          ) : null}
          <div className="grid gap-2">
            <Label htmlFor="username">{t("login.username")}</Label>
            <Input
              id="username"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="password">{t("login.password")}</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <div className="flex justify-end">
              <Button variant="link" className="h-auto px-0 text-xs" asChild>
                <Link to="/forgot-password">{t("login.forgot")}</Link>
              </Button>
            </div>
          </div>
          <AuthChallenge ref={challengeRef} settings={settingsReady ? (settings.data ?? { captchaProvider: "image" }) : undefined} action="login" t={t} />
          <Button type="submit" disabled={pending || !settingsReady} className="h-10 w-full">
            {pending ? t("login.submitting") : t("login.submit")}
          </Button>
          {import.meta.env.DEV ? (
            <p className="text-xs text-muted-foreground">
              admin / admin123 · operator / operator123 · viewer / viewer123
            </p>
          ) : null}
        </form>
      )}
    </GuestChrome>
  )
}
