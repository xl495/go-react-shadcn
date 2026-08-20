import { useState, type FormEvent, type ReactNode } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { toast } from "sonner"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { ApiError } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { useCaptcha } from "@/hooks/queries"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function GuestShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-background">
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
  const [captchaCode, setCaptchaCode] = useState("")
  const [pending, setPending] = useState(false)
  const captcha = useCaptcha()

  if (user) return <Navigate to="/" replace />

  async function refreshCaptcha() {
    setCaptchaCode("")
    await captcha.refetch()
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    try {
      await api.forgotPassword({
        email,
        captchaId: captcha.data?.captchaId ?? "",
        captchaCode,
      })
      toast.success(t("login.forgotSent"))
      await refreshCaptcha()
    } catch (err) {
      toast.error(err instanceof ApiError ? translateApiError(err, t) : t("login.failed"))
      await refreshCaptcha()
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
        <div className="grid gap-2">
          <Label htmlFor="captcha">{t("login.captcha")}</Label>
          <div className="flex items-center gap-3">
            <Input
              id="captcha"
              inputMode="numeric"
              autoComplete="off"
              value={captchaCode}
              onChange={(e) => setCaptchaCode(e.target.value)}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              onClick={() => void refreshCaptcha()}
              className="relative h-9 w-[120px] overflow-hidden p-0"
              aria-label={t("login.refreshCaptcha")}
            >
              {captcha.data?.image ? (
                <img src={captcha.data.image} alt={t("login.captchaAlt")} className="h-full w-full object-cover" />
              ) : (
                <span className="text-xs text-muted-foreground">{t("app.loading")}</span>
              )}
              <RefreshCw className="absolute right-1 bottom-1 size-3 text-foreground/40" />
            </Button>
          </div>
        </div>
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
