import { useState, type FormEvent } from "react"
import { api, ApiError } from "@/lib/api"

export function PasswordPage() {
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
    <form onSubmit={onSubmit} className="grid gap-4 rounded-lg border p-6">
      <h1 className="text-xl font-semibold tracking-tight">修改密码</h1>
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
