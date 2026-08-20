import { useEffect, useState, type FormEvent } from "react"
import { useNavigate } from "react-router-dom"
import { LogOut } from "lucide-react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import type { User } from "@/lib/types"

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
            <MailPrefsForm user={user} onUser={setUser} />
            <ChangePasswordForm />
            {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}
          </section>
        )}
      </main>
    </div>
  )
}

function MailPrefsForm({ user, onUser }: { user: User | null; onUser: (u: User) => void }) {
  const [timezone, setTimezone] = useState(user?.timezone || "Asia/Shanghai")
  const [optIn, setOptIn] = useState(user?.marketingOptIn ?? true)
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)

  if (!user) return null
  const current = user

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setMessage("")
    setError("")
    setPending(true)
    try {
      const next = await api.updateProfile({
        nickname: current.nickname,
        email: current.email ?? "",
        phone: current.phone,
        gender: current.gender ?? "",
        department: current.department ?? "",
        title: current.title ?? "",
        remark: current.remark ?? "",
        timezone,
        marketingOptIn: optIn,
      })
      onUser(next)
      setMessage("已保存")
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "保存失败" : "保存失败")
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-6 grid gap-3 border-t pt-4">
      <p className="text-sm font-medium">邮件偏好</p>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">时区</span>
        <select
          value={timezone}
          onChange={(e) => setTimezone(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {[
            "Asia/Shanghai",
            "Asia/Hong_Kong",
            "Asia/Tokyo",
            "Asia/Singapore",
            "Europe/London",
            "America/New_York",
            "UTC",
          ].map((tz) => (
            <option key={tz} value={tz}>
              {tz}
            </option>
          ))}
        </select>
      </label>
      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={optIn} onChange={(e) => setOptIn(e.target.checked)} />
        接收营销邮件
      </label>
      {message ? <p className="text-sm text-muted-foreground">{message}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="h-9 rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
      >
        {pending ? "保存中…" : "保存偏好"}
      </button>
    </form>
  )
}

function ChangePasswordForm() {
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setMessage("")
    setError("")
    setPending(true)
    try {
      await api.changePassword({ oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      setMessage("密码已更新")
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "修改失败" : "修改失败")
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-6 grid gap-3 border-t pt-4">
      <p className="text-sm font-medium">修改密码</p>
      <input
        type="password"
        autoComplete="current-password"
        placeholder="当前密码"
        value={oldPassword}
        onChange={(e) => setOldPassword(e.target.value)}
        className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
      <input
        type="password"
        autoComplete="new-password"
        placeholder="新密码（至少 8 位）"
        value={newPassword}
        onChange={(e) => setNewPassword(e.target.value)}
        className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
      {message ? <p className="text-sm text-muted-foreground">{message}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="h-9 rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
      >
        {pending ? "保存中…" : "更新密码"}
      </button>
    </form>
  )
}
