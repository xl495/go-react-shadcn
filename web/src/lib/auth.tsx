import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { getToken, readStoredUser, setToken, writeStoredUser } from "@/lib/api"
import type { User } from "@/lib/types"

type AuthState = {
  token: string | null
  user: User | null
  login: (token: string, user: User) => void
  logout: () => void
  setUser: (user: User) => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(() => getToken())
  const [user, setUserState] = useState<User | null>(() => readStoredUser())

  const login = useCallback((nextToken: string, nextUser: User) => {
    setToken(nextToken)
    writeStoredUser(nextUser)
    setTokenState(nextToken)
    setUserState(nextUser)
  }, [])

  const logout = useCallback(() => {
    setToken(null)
    writeStoredUser(null)
    setTokenState(null)
    setUserState(null)
  }, [])

  const setUser = useCallback((next: User) => {
    writeStoredUser(next)
    setUserState(next)
  }, [])

  const value = useMemo<AuthState>(
    () => ({ token, user, login, logout, setUser }),
    [token, user, login, logout, setUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth outside provider")
  return ctx
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { token } = useAuth()
  const location = useLocation()
  if (!token) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return children
}

export function RequireGuest({ children }: { children: ReactNode }) {
  const { token } = useAuth()
  if (token) return <Navigate to="/" replace />
  return children
}
