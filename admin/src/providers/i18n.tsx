import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react"
import { en } from "@/locales/en"
import { zhCN } from "@/locales/zh-CN"
import { zhTW } from "@/locales/zh-TW"
import { ApiError } from "@/api/client"

export const LOCALES = ["zh-CN", "zh-TW", "en"] as const
export type Locale = (typeof LOCALES)[number]

const catalogs: Record<Locale, Record<string, unknown>> = {
  "zh-CN": zhCN,
  "zh-TW": zhTW,
  en,
}

export const LOCALE_META: Record<Locale, { short: string; label: string; html: string }> = {
  "zh-CN": { short: "简", label: "简体中文", html: "zh-CN" },
  "zh-TW": { short: "繁", label: "繁體中文", html: "zh-TW" },
  en: { short: "EN", label: "English", html: "en" },
}

const STORAGE_KEY = "latch.locale"

type I18nState = {
  locale: Locale
  setLocale: (next: Locale) => void
  t: (key: string, vars?: Record<string, string | number>) => string
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

function interpolate(template: string, vars?: Record<string, string | number>) {
  if (!vars) return template
  return template.replace(/\{(\w+)\}/g, (_, name: string) => String(vars[name] ?? `{${name}}`))
}

export function detectLocale(): Locale {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved && (LOCALES as readonly string[]).includes(saved)) return saved as Locale
  const nav = navigator.language || ""
  const lower = nav.toLowerCase()
  if (lower.startsWith("zh-tw") || lower.startsWith("zh-hk") || lower.startsWith("zh-hant")) return "zh-TW"
  if (lower.startsWith("zh")) return "zh-CN"
  return "en"
}

export function applyLocale(locale: Locale) {
  document.documentElement.lang = LOCALE_META[locale].html
  document.title = lookup(catalogs[locale], "app.title") ?? "Latch"
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => detectLocale())

  useEffect(() => {
    applyLocale(locale)
  }, [locale])

  const value = useMemo<I18nState>(() => {
    const t = (key: string, vars?: Record<string, string | number>) => {
      const raw = lookup(catalogs[locale], key) ?? lookup(catalogs["zh-CN"], key) ?? key
      return interpolate(raw, vars)
    }
    return {
      locale,
      setLocale: (next) => {
        localStorage.setItem(STORAGE_KEY, next)
        setLocaleState(next)
      },
      t,
    }
  }, [locale])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error("useI18n outside provider")
  return ctx
}

export function useT() {
  return useI18n().t
}

export function translateApiError(err: unknown, t: I18nState["t"]) {
  if (err instanceof ApiError) {
    const mapped = t(`errors.${err.code}`)
    if (mapped !== `errors.${err.code}`) return mapped
    return err.message || t("errors.fallback")
  }
  if (err instanceof Error && err.message) return err.message
  return t("errors.fallback")
}

export function roleLabel(code: string, fallback: string, t: I18nState["t"]) {
  const key = `roles.${code}`
  const translated = t(key)
  return translated === key ? fallback : translated
}

export function permLabel(code: string, fallback: string, t: I18nState["t"]) {
  const key = `perm.${code}`
  const translated = t(key)
  return translated === key ? fallback : translated
}

export function roleDesc(code: string, fallback: string, t: I18nState["t"]) {
  const key = `roles.${code}Desc`
  const translated = t(key)
  return translated === key ? fallback : translated
}

