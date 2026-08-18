import { createContext, useContext, useMemo, useState, type ReactNode } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { setToken } from "@/lib/api"
import type { User } from "@/lib/types"

const USER_KEY = "latch.user"

type AuthState = {
  user: User | null
  login: (token: string, user: User) => void
  logout: () => void
  updateUser: (next: User) => void
  isAdmin: boolean
  can: (code: string) => boolean
}

const AuthContext = createContext<AuthState | null>(null)

function readUser(): User | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(readUser)

  const value = useMemo<AuthState>(() => {
    const codes = new Set<string>(user?.permissionCodes ?? [])
    for (const r of user?.roles ?? []) {
      for (const p of r.permissions ?? []) codes.add(p.code)
    }
    const roleCodes = new Set((user?.roles ?? []).map((r) => r.code))
    const isAdmin = roleCodes.has("admin") || codes.has("admin:all") || codes.has("*")
    return {
      user,
      isAdmin,
      can: (code) => isAdmin || codes.has(code),
      login: (token, next) => {
        setToken(token)
        localStorage.setItem(USER_KEY, JSON.stringify(next))
        setUser(next)
      },
      logout: () => {
        setToken(null)
        localStorage.removeItem(USER_KEY)
        setUser(null)
      },
      updateUser: (next) => {
        localStorage.setItem(USER_KEY, JSON.stringify(next))
        setUser(next)
      },
    }
  }, [user])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth outside provider")
  return ctx
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const location = useLocation()
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return children
}

export function RequirePerm({ perm, children }: { perm: string; children: ReactNode }) {
  const { can } = useAuth()
  if (!can(perm)) return <Navigate to="/" replace />
  return children
}

