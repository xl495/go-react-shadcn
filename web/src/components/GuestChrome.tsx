import type { ReactNode } from "react"
import { LanguageSwitcher } from "@/components/LanguageSwitcher"
import { ThemeToggle } from "@/components/ThemeToggle"
import { useI18n } from "@/lib/i18n"

export function GuestChrome({ children, trailing }: { children: ReactNode; trailing?: ReactNode }) {
  const { t } = useI18n()
  return (
    <div className="flex h-full overflow-hidden bg-background text-foreground">
      <aside className="relative hidden min-h-0 w-[min(52%,38rem)] shrink-0 lg:block">
        <img src="/auth-hero.jpg" alt="" fetchPriority="high" decoding="async" className="absolute inset-0 size-full object-cover" />
        <div className="absolute inset-0 bg-gradient-to-t from-[#1c1814] via-[#1c1814]/20 to-[#1c1814]/35" />
        <div className="relative flex h-full flex-col justify-between p-10 text-[#f4eadc]">
          <span className="font-display text-3xl tracking-tight">gra</span>
          <div className="max-w-sm">
            <p className="font-display text-[2.55rem] leading-[1.05]">{t("login.panelTitle")}</p>
            <p className="mt-4 text-sm/6 text-[#f4eadc]/70">{t("login.panelHint")}</p>
          </div>
        </div>
      </aside>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between px-5 sm:px-8">
          <span className="flex items-center gap-2.5 lg:hidden">
            <img src="/gra-mark.png" alt="" width={32} height={32} className="size-8 rounded-md" />
            <span className="font-display text-xl leading-none tracking-tight">gra</span>
          </span>
          <div className="ml-auto flex items-center gap-3">
            {trailing}
            <ThemeToggle />
            <LanguageSwitcher />
          </div>
        </header>
        <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-5 py-8 sm:px-8">
          <div className="auth-rise w-full max-w-[22rem]">{children}</div>
        </div>
      </div>
    </div>
  )
}
