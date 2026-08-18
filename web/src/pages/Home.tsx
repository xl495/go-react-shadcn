import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { LogOut } from "lucide-react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"

export function HomePage() {
  const { user, setUser, logout } = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(!user)

  useEffect(() => {
    let cancelled = false
    api
      .me()
      .then((me) => {
        if (!cancelled) {
          setUser(me)
          setError("")
          setLoading(false)
        }
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof ApiError && (err.status === 401 || err.code === 40101 || err.code === 40102)) {
          logout()
          navigate("/login", { replace: true })
          return
        }
        setError(err instanceof ApiError ? err.message : "加载用户信息失败")
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [logout, navigate, setUser])

  const displayName = user?.nickname || user?.username || "—"
  const initial = displayName.slice(0, 1).toUpperCase()

  function onSignOut() {
    logout()
    navigate("/login", { replace: true })
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="flex h-14 items-center justify-between border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
        <button
          type="button"
          onClick={onSignOut}
          className="inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm hover:bg-muted"
        >
          <LogOut className="size-4" />
          退出登录
        </button>
      </header>
      <main className="mx-auto flex w-full max-w-lg flex-1 flex-col justify-center px-4 py-10">
        {loading && !user ? (
          <p className="text-sm text-muted-foreground">加载中…</p>
        ) : (
          <section className="rounded-lg border p-6">
            <div className="flex items-center gap-4">
              {user?.avatar ? (
                <img
                  src={user.avatar}
                  alt={displayName}
                  className="size-16 rounded-full border object-cover"
                />
              ) : (
                <div className="flex size-16 items-center justify-center rounded-full border bg-muted text-xl font-semibold">
                  {initial}
                </div>
              )}
              <div className="min-w-0">
                <h1 className="truncate text-xl font-semibold tracking-tight">{displayName}</h1>
                {user?.nickname && user.username ? (
                  <p className="truncate text-sm text-muted-foreground">@{user.username}</p>
                ) : null}
              </div>
            </div>
            <dl className="mt-6 grid gap-3 text-sm">
              <div className="flex justify-between gap-4 border-t pt-3">
                <dt className="text-muted-foreground">用户名</dt>
                <dd>{user?.username || "—"}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">手机号</dt>
                <dd>{user?.phone || "未绑定"}</dd>
              </div>
            </dl>
            {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
          </section>
        )}
      </main>
    </div>
  )
}
