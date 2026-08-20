import { LOCALE_META, LOCALES, useI18n, type Locale } from "@/providers/i18n"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  return (
    <Select value={locale} onValueChange={(value) => setLocale(value as Locale)}>
      <SelectTrigger aria-label={t("app.language")} className="w-38">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {LOCALES.map((item) => (
          <SelectItem key={item} value={item}>
            {LOCALE_META[item].label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
