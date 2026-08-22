import { createContext, useContext, useMemo, useState, type ReactNode } from "react"
import { en } from "@/locales/en"
import { zhCN, type WebMessages } from "@/locales/zh-CN"

export const LOCALES = ["zh-CN", "en"] as const
export type Locale = (typeof LOCALES)[number]

const STORAGE_KEY = "latch.web.locale"

export const LOCALE_META: Record<Locale, { short: string; label: string; html: string }> = {
  "zh-CN": { short: "中", label: "简体中文", html: "zh-CN" },
  en: { short: "EN", label: "English", html: "en" },
}

const catalogs: Record<Locale, WebMessages> = {
  "zh-CN": zhCN,
  en,
}

type I18nState = {
  locale: Locale
  setLocale: (next: Locale) => void
  t: (key: string) => string
}

const I18nContext = createContext<I18nState | null>(null)

function lookup(tree: unknown, key: string): string | undefined {
  let cur: unknown = tree
  for (const part of key.split(".")) {
    if (!cur || typeof cur !== "object" || !(part in cur)) return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  return typeof cur === "string" ? cur : undefined
}

export function detectLocale(): Locale {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved && (LOCALES as readonly string[]).includes(saved)) return saved as Locale
  const nav = (navigator.language || "").toLowerCase()
  if (nav.startsWith("zh")) return "zh-CN"
  return "en"
}

function applyLocale(locale: Locale) {
  document.documentElement.lang = LOCALE_META[locale].html
  document.title = catalogs[locale].app.title
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const next = detectLocale()
    applyLocale(next)
    return next
  })

  const value = useMemo<I18nState>(
    () => ({
      locale,
      setLocale: (next) => {
        localStorage.setItem(STORAGE_KEY, next)
        applyLocale(next)
        setLocaleState(next)
      },
      t: (key) => lookup(catalogs[locale], key) ?? lookup(zhCN, key) ?? key,
    }),
    [locale],
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error("useI18n outside provider")
  return ctx
}

const MENU_NAV_KEYS: Record<string, string> = {
  "web:home": "nav.home",
  "web:profile": "nav.profile",
  "web:password": "nav.password",
  "web:devices": "nav.devices",
  "web:notify": "nav.notifications",
}

export function menuLabel(code: string, fallback: string, t: I18nState["t"]) {
  const navKey = MENU_NAV_KEYS[code]
  if (navKey) {
    const translated = t(navKey)
    if (translated !== navKey) return translated
  }
  return fallback
}
