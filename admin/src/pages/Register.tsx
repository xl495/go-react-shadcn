import { Navigate, Link } from "react-router-dom"
import { GuestChrome } from "@/components/layout/GuestChrome"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { PageFallback } from "@/components/PageFallback"
import { Button } from "@/components/ui/button"

export function RegisterPage() {
  const { user, loading } = useAuth()
  const { t } = useI18n()

  if (loading) return <PageFallback />
  if (user) return <Navigate to="/" replace />

  return (
    <GuestChrome>
      <div>
        <h1 className="font-display text-[2rem] leading-none tracking-tight">{t("login.registerTitle")}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t("login.registerSubtitle")}</p>
      </div>
      <Button variant="link" className="h-auto px-0" asChild>
        <Link to="/login">{t("login.backToLogin")}</Link>
      </Button>
    </GuestChrome>
  )
}
