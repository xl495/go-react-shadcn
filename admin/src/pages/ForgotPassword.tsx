import { useRef, useState, type FormEvent, type ReactNode } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { ApiError } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { useAuthSettings } from "@/hooks/queries"
import { api } from "@/api/client"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/auth/AuthChallenge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

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

export function ForgotPasswordPage() {
  const { user } = useAuth()
  const { t } = useI18n()
  const [email, setEmail] = useState("")
  const [pending, setPending] = useState(false)
  const settings = useAuthSettings()
  const challengeRef = useRef<AuthChallengeHandle>(null)

  if (user) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    try {
      const challenge = (await challengeRef.current?.collect()) ?? {}
      await api.forgotPassword({ email, ...challenge })
      toast.success(t("login.forgotSent"))
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      toast.error(err instanceof ApiError ? translateApiError(err, t) : t("login.failed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <GuestShell>
      <form onSubmit={onSubmit} className="space-y-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("login.forgotTitle")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("login.forgotSubtitle")}</p>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="email">{t("login.email")}</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <AuthChallenge ref={challengeRef} settings={settings.data} action="forgot" t={t} />
        <Button type="submit" disabled={pending} className="h-10 w-full">
          {pending ? t("login.forgotSubmitting") : t("login.forgotSubmit")}
        </Button>
        <Button variant="link" className="h-auto px-0" asChild>
          <Link to="/login">{t("login.backToLogin")}</Link>
        </Button>
      </form>
    </GuestShell>
  )
}

export function ResetPasswordPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const token = new URLSearchParams(window.location.search).get("token") ?? ""
  const [password, setPassword] = useState("")
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token) {
      toast.error(t("login.resetMissingToken"))
      return
    }
    setPending(true)
    try {
      await api.resetPassword({ token, newPassword: password })
      toast.success(t("login.resetDone"))
      navigate("/login", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? translateApiError(err, t) : t("login.failed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <GuestShell>
      <form onSubmit={onSubmit} className="space-y-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("login.resetTitle")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("login.resetSubtitle")}</p>
        </div>
        {!token ? <p className="text-sm text-destructive">{t("login.resetMissingToken")}</p> : null}
        <div className="grid gap-2">
          <Label htmlFor="np">{t("settings.newPassword")}</Label>
          <Input
            id="np"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <Button type="submit" disabled={pending || !token} className="h-10 w-full">
          {pending ? t("login.resetSubmitting") : t("login.resetSubmit")}
        </Button>
        <Button variant="link" className="h-auto px-0" asChild>
          <Link to="/login">{t("login.backToLogin")}</Link>
        </Button>
      </form>
    </GuestShell>
  )
}
