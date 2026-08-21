const KEY = "latch.theme"

export type Theme = "light" | "dark" | "system"

export function readTheme(): Theme {
  const raw = localStorage.getItem(KEY)
  if (raw === "light" || raw === "dark" || raw === "system") return raw
  return "system"
}

export function applyTheme(theme: Theme) {
  const dark =
    theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches)
  document.documentElement.classList.toggle("dark", dark)
  document.documentElement.style.colorScheme = dark ? "dark" : "light"
}

export function cycleTheme(theme: Theme): Theme {
  return theme === "system" ? "light" : theme === "light" ? "dark" : "system"
}

export function persistTheme(theme: Theme) {
  localStorage.setItem(KEY, theme)
  applyTheme(theme)
}
