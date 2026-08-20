import { Check, Globe } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { LOCALE_META, LOCALES, useI18n, type Locale } from "@/providers/i18n"

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  const current = LOCALE_META[locale]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-9 gap-1.5 px-2.5"
          aria-label={`${t("app.language")}: ${current.label}`}
        >
          <Globe className="size-3.5" />
          <span className="min-w-4 text-xs font-semibold tracking-wide">{current.short}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-40">
        {LOCALES.map((item) => (
          <DropdownMenuItem
            key={item}
            onSelect={() => setLocale(item as Locale)}
            aria-current={item === locale ? "true" : undefined}
          >
            <span className="w-6 text-xs font-semibold tracking-wide">{LOCALE_META[item].short}</span>
            <span className="flex-1">{LOCALE_META[item].label}</span>
            {item === locale ? <Check className="size-3.5" /> : <span className="size-3.5" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
