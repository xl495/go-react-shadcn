import { Monitor, Moon, Sun } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/providers/i18n"
import { cycleTheme, useTheme } from "@/providers/theme"

export function ThemeToggle() {
  const { theme, resolved, setTheme } = useTheme()
  const { t } = useI18n()
  const Icon = theme === "system" ? Monitor : resolved === "dark" ? Moon : Sun
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={t("nav.theme")}
      title={t(`nav.themeMode.${theme}`)}
      onClick={() => setTheme(cycleTheme(theme))}
    >
      <Icon className="size-4" />
    </Button>
  )
}
