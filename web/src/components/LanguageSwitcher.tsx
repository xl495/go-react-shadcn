import { LOCALE_META, LOCALES, useI18n, type Locale } from "@/lib/i18n"

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  return (
    <div className="flex items-center gap-1" role="group" aria-label={t("app.language")}>
      {LOCALES.map((item) => (
        <button
          key={item}
          type="button"
          onClick={() => setLocale(item as Locale)}
          aria-pressed={item === locale}
          className={`inline-flex h-8 min-w-8 items-center justify-center rounded-md px-2 text-xs font-semibold ${
            item === locale ? "bg-muted" : "hover:bg-muted"
          }`}
        >
          {LOCALE_META[item].short}
        </button>
      ))}
    </div>
  )
}
