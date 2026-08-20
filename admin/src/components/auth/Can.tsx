import type { ReactNode } from "react"
import { useAuth } from "@/providers/auth"

export function Can({
  perm,
  children,
  fallback = null,
}: {
  perm: string | string[]
  children: ReactNode
  fallback?: ReactNode
}) {
  const { can } = useAuth()
  const ok = Array.isArray(perm) ? perm.some((p) => can(p)) : can(perm)
  return ok ? children : fallback
}
