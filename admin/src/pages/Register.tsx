import { Navigate, useNavigate } from "react-router-dom"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { ApiError } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { translateApiError, useI18n } from "@/providers/i18n"
import { useAuthSettings, useGoogleAuthMutation } from "@/hooks/queries"
import { PageFallback } from "@/components/PageFallback"
import { GoogleSignIn } from "@/components/auth/GoogleSignIn"
import { Button } from "@/components/ui/button"

export function RegisterPage() {
  const { user, loading, login } = useAuth()
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const settings = useAuthSettings()
  const googleMut = useGoogleAuthMutation()

  if (loading) return <PageFallback />
  if (user) return <Navigate to="/" replace />

  async function onGoogle(idToken: string) {
    try {
      const result = await googleMut.mutateAsync({ idToken, client: "admin" })
      login(result.token, result.user)
      toast.success(t("login.registerDone"))
      navigate("/", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? translateApiError(err, t) : t("login.failed"))
    }
  }

  return (
    <div className="flex h-full flex-col overflow-y-auto bg-background">
      <header className="flex h-14 items-center justify-between border-b px-6">
        <span className="text-sm font-semibold">Latch</span>
        <LanguageSwitcher />
      </header>
      <div className="flex flex-1 items-center justify-center px-4">
        <div className="w-full max-w-sm space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("login.registerTitle")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("login.registerSubtitle")}</p>
          </div>
          {settings.data && !settings.data.googleEnabled ? (
            <p className="text-sm text-muted-foreground">{t("login.googleDisabled")}</p>
          ) : null}
          {settings.data?.googleEnabled && !settings.data.googleRegisterEnabled ? (
            <p className="text-sm text-muted-foreground">{t("login.registerClosed")}</p>
          ) : null}
          {settings.data?.googleEnabled && settings.data.googleRegisterEnabled ? (
            <GoogleSignIn
              clientId={settings.data.googleClientId}
              locale={locale}
              onCredential={onGoogle}
              disabled={googleMut.isPending}
            />
          ) : null}
          <Button variant="link" className="h-auto px-0" asChild>
            <Link to="/login">{t("login.backToLogin")}</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}
