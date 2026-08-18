import { LOCALE_META, LOCALES, useI18n, type Locale } from "@/lib/i18n"
import { cn } from "@/lib/utils"

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  return (
    <div
      role="group"
      aria-label={t("app.language")}
      className="inline-flex rounded-md border border-border p-0.5 text-xs text-muted-foreground"
    >
      {LOCALES.map((item) => (
        <button
          key={item}
          type="button"
          onClick={() => setLocale(item as Locale)}
          className={cn(
            "rounded px-2 py-1 transition-colors",
            locale === item ? "bg-foreground text-background" : "hover:text-foreground",
          )}
        >
          {LOCALE_META[item].short}
        </button>
      ))}
    </div>
  )
}
