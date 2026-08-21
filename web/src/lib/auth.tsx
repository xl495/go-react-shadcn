import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react"
import { Navigate, useLocation } from "react-router-dom"
import {
  api,
  getToken,
  isSessionExpired,
  readStoredUser,
  setToken,
  setUnauthorizedHandler,
  writeStoredUser,
} from "@/lib/api"
import { useI18n } from "@/lib/i18n"
import type { User } from "@/lib/types"

type AuthState = {
  token: string | null
  user: User | null
  loading: boolean
  login: (token: string, user: User) => void
  logout: () => Promise<void>
  setUser: (user: User) => void
}

const AuthContext = createContext<AuthState | null>(null)

function clearSession() {
  setToken(null)
  writeStoredUser(null)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(() => getToken())
  const [user, setUserState] = useState<User | null>(() => readStoredUser())
  const [bootstrapping, setBootstrapping] = useState(() => !!getToken())

  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearSession()
      setTokenState(null)
      setUserState(null)
      setBootstrapping(false)
      if (!window.location.pathname.startsWith("/login")) {
        window.location.assign("/login")
      }
    })
    return () => setUnauthorizedHandler(null)
  }, [])

  useEffect(() => {
    if (!token) return
    let cancelled = false
    api
      .me()
      .then((me) => {
        if (cancelled) return
        writeStoredUser(me)
        setUserState(me)
        setBootstrapping(false)
      })
      .catch((err) => {
        if (cancelled) return
        if (isSessionExpired(err)) {
          clearSession()
          setTokenState(null)
          setUserState(null)
        }
        setBootstrapping(false)
      })
    return () => {
      cancelled = true
    }
  }, [token])

  const login = useCallback((nextToken: string, nextUser: User) => {
    setToken(nextToken)
    writeStoredUser(nextUser)
    setTokenState(nextToken)
    setUserState(nextUser)
  }, [])

  const logout = useCallback(async () => {
    try {
      if (getToken()) await api.logout()
    } catch {
      /* still clear local session */
    }
    clearSession()
    setTokenState(null)
    setUserState(null)
    setBootstrapping(false)
  }, [])

  const setUser = useCallback((next: User) => {
    writeStoredUser(next)
    setUserState(next)
  }, [])

  const loading = bootstrapping && !user
  const value = useMemo<AuthState>(
    () => ({ token, user, loading, login, logout, setUser }),
    [token, user, loading, login, logout, setUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth outside provider")
  return ctx
}

function AuthFallback() {
  const { t } = useI18n()
  return <p className="p-8 text-sm text-muted-foreground">{t("app.loading")}</p>
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { token, user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <AuthFallback />
  if (!token || !user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return children
}

export function RequireGuest({ children }: { children: ReactNode }) {
  const { token, user, loading } = useAuth()
  if (loading) return <AuthFallback />
  if (token && user) return <Navigate to="/" replace />
  return children
}
