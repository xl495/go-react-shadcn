import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react"

const KEY = "latch.theme"

export type Theme = "light" | "dark" | "system"
export type ResolvedTheme = "light" | "dark"

type ThemeState = {
  theme: Theme
  resolved: ResolvedTheme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeState | null>(null)

function readTheme(): Theme {
  const raw = localStorage.getItem(KEY)
  if (raw === "light" || raw === "dark" || raw === "system") return raw
  return "system"
}

export function resolvedTheme(theme: Theme, systemDark: boolean): ResolvedTheme {
  if (theme === "light" || theme === "dark") return theme
  return systemDark ? "dark" : "light"
}

export function cycleTheme(theme: Theme): Theme {
  return theme === "system" ? "light" : theme === "light" ? "dark" : "system"
}

export function applyTheme(theme: Theme, systemDark = window.matchMedia("(prefers-color-scheme: dark)").matches) {
  const dark = resolvedTheme(theme, systemDark) === "dark"
  document.documentElement.classList.toggle("dark", dark)
  document.documentElement.style.colorScheme = dark ? "dark" : "light"
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(readTheme)
  const [systemDark, setSystemDark] = useState(
    () => window.matchMedia("(prefers-color-scheme: dark)").matches,
  )

  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const onChange = () => setSystemDark(mq.matches)
    mq.addEventListener("change", onChange)
    return () => mq.removeEventListener("change", onChange)
  }, [])

  const resolved = resolvedTheme(theme, systemDark)

  useEffect(() => {
    applyTheme(theme, systemDark)
    localStorage.setItem(KEY, theme)
  }, [theme, systemDark])

  const value = useMemo<ThemeState>(
    () => ({
      theme,
      resolved,
      setTheme: (next) => setThemeState(next),
    }),
    [theme, resolved],
  )
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error("useTheme")
  return ctx
}
