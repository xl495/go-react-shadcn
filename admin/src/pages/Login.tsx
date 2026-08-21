import { useRef, useState, type FormEvent } from "react"
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { GuestChrome } from "@/components/layout/GuestChrome"
import { ApiError } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { useAuthSettings, useGoogleAuthMutation, useLoginMutation } from "@/hooks/queries"
import { PageFallback } from "@/components/PageFallback"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/auth/AuthChallenge"
import { GoogleSignIn } from "@/components/auth/GoogleSignIn"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { safeInternalPath } from "@latch/auth/safe-path"

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
  const pending = loginMut.isPending || googleMut.isPending

  if (loading) return <PageFallback />
  if (user) return <Navigate to="/" replace />

  function finish(token: string, nextUser: typeof user) {
    if (!nextUser) return
    login(token, nextUser)
    toast.success(t("login.submit"))
    navigate(from, { replace: true })
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
      finish(result.token, result.user)
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
      finish(result.token, result.user)
    } catch (err) {
      if (!(err instanceof ApiError)) toast.error(t("login.failed"))
    }
  }

  return (
    <GuestChrome>
        <form onSubmit={onSubmit} className="space-y-5">
          <div>
            <p className="font-display text-[13px] text-muted-foreground">gra</p>
            <h1 className="mt-1 font-display text-[2rem] leading-none tracking-tight">{t("login.title")}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{t("login.subtitle")}</p>
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
    </GuestChrome>
  )
}
