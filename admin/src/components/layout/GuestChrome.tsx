import type { ReactNode } from "react"
import { GraMark, GraWord } from "@/components/layout/Brand"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { ThemeToggle } from "@/components/layout/ThemeToggle"
import { useI18n } from "@/providers/i18n"

export function GuestChrome({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  return (
    <div className="flex h-full overflow-hidden bg-background text-foreground">
      <aside className="relative hidden min-h-0 w-[min(52%,38rem)] shrink-0 lg:block">
        <img src="/auth-hero.jpg" alt="" fetchPriority="high" decoding="async" className="absolute inset-0 size-full object-cover" />
        <div className="absolute inset-0 bg-gradient-to-t from-[#1c1814] via-[#1c1814]/20 to-[#1c1814]/35" />
        <div className="relative flex h-full flex-col justify-between p-10 text-[#f4eadc]">
          <GraWord className="text-3xl" />
          <div className="max-w-sm">
            <p className="font-display text-[2.55rem] leading-[1.05]">{t("login.panelTitle")}</p>
            <p className="mt-4 text-sm/6 text-[#f4eadc]/70">{t("login.panelHint")}</p>
          </div>
        </div>
      </aside>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between px-5 sm:px-8">
          <span className="flex items-center gap-2.5 lg:hidden">
            <GraMark className="size-8" />
            <GraWord className="text-xl leading-none" />
          </span>
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            <LanguageSwitcher />
          </div>
        </header>
        <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-5 py-8 sm:px-8">
          <div className="auth-rise w-full max-w-[22rem] space-y-5">{children}</div>
        </div>
      </div>
    </div>
  )
}
