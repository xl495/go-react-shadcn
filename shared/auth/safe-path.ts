export function safeInternalPath(raw: unknown, fallback = "/"): string {
  if (typeof raw !== "string" || !raw.startsWith("/") || raw.startsWith("//") || raw.includes("\\")) {
    return fallback
  }
  if (raw.startsWith("/login") || raw.startsWith("/forgot-password") || raw.startsWith("/register") || raw.startsWith("/reset-password")) {
    return fallback
  }
  return raw
}
