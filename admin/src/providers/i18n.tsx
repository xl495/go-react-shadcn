import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react"
import { ApiError } from "@/api/client"

export const LOCALES = ["zh-CN", "en"] as const
export type Locale = (typeof LOCALES)[number]

const loaders: Record<Locale, () => Promise<Record<string, unknown>>> = {
  "zh-CN": () => import("@/locales/zh-CN").then((m) => m.zhCN as unknown as Record<string, unknown>),
  en: () => import("@/locales/en").then((m) => m.en as unknown as Record<string, unknown>),
}

export const LOCALE_META: Record<Locale, { short: string; label: string; html: string }> = {
  "zh-CN": { short: "中", label: "简体中文", html: "zh-CN" },
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
  if (saved === "zh-TW") return "zh-CN"
  if (saved && (LOCALES as readonly string[]).includes(saved)) return saved as Locale
  const nav = navigator.language || ""
  const lower = nav.toLowerCase()
  if (lower.startsWith("zh")) return "zh-CN"
  return "en"
}

export function applyLocale(locale: Locale, catalog?: Record<string, unknown>) {
  document.documentElement.lang = LOCALE_META[locale].html
  document.title = (catalog ? lookup(catalog, "app.title") : undefined) ?? "gra"
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => detectLocale())
  const [catalog, setCatalog] = useState<Record<string, unknown> | null>(null)
  const [fallback, setFallback] = useState<Record<string, unknown> | null>(null)

  useEffect(() => {
    let cancelled = false
    loaders[locale]().then((next) => {
      if (cancelled) return
      setCatalog(next)
      applyLocale(locale, next)
    })
    return () => {
      cancelled = true
    }
  }, [locale])

  useEffect(() => {
    if (locale === "zh-CN") return
    let cancelled = false
    loaders["zh-CN"]().then((next) => {
      if (!cancelled) setFallback(next)
    })
    return () => {
      cancelled = true
    }
  }, [locale])

  const value = useMemo<I18nState>(() => {
    const t = (key: string, vars?: Record<string, string | number>) => {
      const raw = lookup(catalog, key) ?? lookup(fallback, key) ?? key
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
  }, [locale, catalog, fallback])

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
    if (err.message) return err.message
    const mapped = t(`errors.${err.code}`)
    if (mapped !== `errors.${err.code}`) return mapped
    return t("errors.fallback")
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

const MENU_NAV_KEYS: Record<string, string> = {
  "dashboard:read": "nav.dashboard",
  "org:menu": "nav.org",
  "user:list": "nav.users",
  "webuser:list": "nav.webUsers",
  "dept:list": "nav.departments",
  "role:list": "nav.roles",
  "perm:list": "nav.permissions",
  "system:menu": "nav.system",
  "dict:list": "nav.dicts",
  "config:list": "nav.configs",
  "mail:jobs:list": "nav.mailJobs",
  "mail:campaign:list": "nav.mailCampaigns",
  "log:list": "nav.logs",
  "menu:list": "nav.menus",
  "notify:list": "nav.notifications",
  "session:list": "nav.online",
}

export function menuLabel(code: string, fallback: string, t: I18nState["t"]) {
  const navKey = MENU_NAV_KEYS[code]
  if (navKey) {
    const translated = t(navKey)
    if (translated !== navKey) return translated
  }
  const key = `menu.${code}`
  const translated = t(key)
  return translated === key ? fallback : translated
}

export function roleDesc(code: string, fallback: string, t: I18nState["t"]) {
  const key = `roles.${code}Desc`
  const translated = t(key)
  return translated === key ? fallback : translated
}
