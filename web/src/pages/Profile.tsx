import { useEffect, useState, type FormEvent } from "react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"

export function ProfilePage() {
  const { user, setUser } = useAuth()
  const [nickname, setNickname] = useState(user?.nickname || "")
  const [email, setEmail] = useState(user?.email || "")
  const [phone, setPhone] = useState(user?.phone || "")
  const [timezone, setTimezone] = useState(user?.timezone || "Asia/Shanghai")
  const [optIn, setOptIn] = useState(user?.marketingOptIn ?? true)
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (!user) return
    setNickname(user.nickname || "")
    setEmail(user.email || "")
    setPhone(user.phone || "")
    setTimezone(user.timezone || "Asia/Shanghai")
    setOptIn(user.marketingOptIn ?? true)
  }, [user])

  if (!user) return null
  const current = user

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setMessage("")
    setError("")
    setPending(true)
    try {
      const next = await api.updateProfile({
        nickname,
        email,
        phone,
        gender: current.gender ?? "",
        department: current.department ?? "",
        title: current.title ?? "",
        remark: current.remark ?? "",
        timezone,
        marketingOptIn: optIn,
      })
      setUser(next)
      setMessage("已保存")
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "保存失败" : "保存失败")
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="grid gap-4 rounded-lg border p-6">
      <h1 className="text-xl font-semibold tracking-tight">我的资料</h1>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">昵称</span>
        <input
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">邮箱</span>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">手机号</span>
        <input
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
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
        {pending ? "保存中…" : "保存资料"}
      </button>
    </form>
  )
}
