import { Link, useLocation } from "react-router-dom"
import { useI18n } from "@/lib/i18n"
import { Button } from "@/components/ui/button"

export function ForbiddenPage() {
  const { t } = useI18n()
  const location = useLocation()
  const perm = (location.state as { perm?: string } | null)?.perm
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 text-center">
      <p className="text-6xl font-semibold text-muted-foreground">403</p>
      <h2 className="text-lg font-medium">{t("errors.forbiddenTitle")}</h2>
      <p className="max-w-md text-sm text-muted-foreground">
        {perm ? t("errors.forbiddenPerm", { perm }) : t("errors.forbiddenBody")}
      </p>
      <Button variant="outline" onClick={() => window.history.back()}>
        {t("errors.goBack")}
      </Button>
    </div>
  )
}

export function NotFoundPage() {
  const { t } = useI18n()
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 text-center">
      <p className="text-6xl font-semibold text-muted-foreground">404</p>
      <h2 className="text-lg font-medium">{t("errors.notFoundTitle")}</h2>
      <p className="text-sm text-muted-foreground">{t("errors.notFoundBody")}</p>
      <Button variant="outline" asChild>
        <Link to="/">{t("errors.goHome")}</Link>
      </Button>
    </div>
  )
}
