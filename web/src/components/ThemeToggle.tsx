import { Monitor, Moon, Sun } from "lucide-react"
import { useI18n } from "@/lib/i18n"
import { cycleTheme, useTheme } from "@/providers/theme"

export function ThemeToggle() {
  const { t } = useI18n()
  const { theme, resolved, setTheme } = useTheme()
  const Icon = theme === "system" ? Monitor : resolved === "dark" ? Moon : Sun
  const modeKey =
    theme === "dark" ? "nav.themeDark" : theme === "light" ? "nav.themeLight" : "nav.themeSystem"
  return (
    <button
      type="button"
      aria-label={t("nav.theme")}
      title={t(modeKey)}
      className="inline-flex size-9 items-center justify-center rounded-md text-muted-foreground hover:bg-muted"
      onClick={() => setTheme(cycleTheme(theme))}
    >
      <Icon className="size-4" />
    </button>
  )
}
