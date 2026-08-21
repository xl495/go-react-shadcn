import { Link } from "react-router-dom"
import { useI18n } from "@/lib/i18n"

export function NotFoundPage() {
  const { t } = useI18n()
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 p-8 text-center">
      <p className="text-6xl font-semibold text-muted-foreground">404</p>
      <h2 className="text-lg font-medium">{t("errors.notFoundTitle")}</h2>
      <p className="text-sm text-muted-foreground">{t("errors.notFoundBody")}</p>
      <Link
        to="/"
        className="h-9 rounded-md border px-3 text-sm leading-9 text-foreground underline-offset-4 hover:underline"
      >
        {t("errors.goHome")}
      </Link>
    </div>
  )
}
