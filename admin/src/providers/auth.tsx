import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { api, getToken, setToken, setUnauthorizedHandler, isSessionExpired } from "@/api/client"
import { useMe } from "@/hooks/queries"
import { queryClient } from "@/providers/query-client"
import { PageFallback } from "@/components/PageFallback"
import type { User } from "@/types"

const USER_KEY = "latch.user"

type AuthState = {
  user: User | null
  loading: boolean
  login: (token: string, user: User) => void
  logout: () => Promise<void>
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

function isAuthError(err: unknown) {
  return isSessionExpired(err)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [cachedUser, setCachedUser] = useState<User | null>(readUser)
  const hasToken = !!getToken()
  const meQuery = useMe(hasToken)
  const user = isAuthError(meQuery.error) ? null : (meQuery.data ?? (hasToken ? cachedUser : null))
  const loading = hasToken && meQuery.isPending && !user

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setToken(null)
      localStorage.removeItem(USER_KEY)
      setCachedUser(null)
      queryClient.removeQueries({ queryKey: ["auth", "me"] })
      if (!window.location.pathname.startsWith("/login")) {
        window.location.assign("/login")
      }
    })
    return () => setUnauthorizedHandler(null)
  }, [])

  useEffect(() => {
    if (meQuery.data) {
      localStorage.setItem(USER_KEY, JSON.stringify(meQuery.data))
    }
  }, [meQuery.data])

  useEffect(() => {
    if (!isAuthError(meQuery.error)) return
    setToken(null)
    localStorage.removeItem(USER_KEY)
  }, [meQuery.error])

  const value = useMemo<AuthState>(() => {
    const codes = new Set<string>(user?.permissionCodes ?? [])
    for (const r of user?.roles ?? []) {
      for (const p of r.permissions ?? []) codes.add(p.code)
    }
    const roleCodes = new Set((user?.roles ?? []).map((r) => r.code))
    const isAdmin = roleCodes.has("admin") || codes.has("admin:all") || codes.has("*")
    return {
      user,
      loading,
      isAdmin,
      can: (code) => isAdmin || codes.has(code),
      login: (token, next) => {
        setToken(token)
        localStorage.setItem(USER_KEY, JSON.stringify(next))
        setCachedUser(next)
        queryClient.setQueryData(["auth", "me"], next)
      },
      logout: async () => {
        try {
          if (getToken()) await api.logout()
        } catch {
          /* still clear local session */
        }
        setToken(null)
        localStorage.removeItem(USER_KEY)
        setCachedUser(null)
        queryClient.clear()
      },
      updateUser: (next) => {
        localStorage.setItem(USER_KEY, JSON.stringify(next))
        setCachedUser(next)
        queryClient.setQueryData(["auth", "me"], next)
      },
    }
  }, [user, loading])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth outside provider")
  return ctx
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <PageFallback />
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (user.mustChangePassword && !location.pathname.startsWith("/settings/password")) {
    return <Navigate to="/settings/password" replace />
  }
  return children
}

export function RequirePerm({ perm, children }: { perm: string; children: ReactNode }) {
  const { can } = useAuth()
  const location = useLocation()
  if (!can(perm)) return <Navigate to="/403" replace state={{ perm, from: location.pathname }} />
  return children
}
