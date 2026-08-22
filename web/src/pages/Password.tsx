import { useState, type FormEvent } from "react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"

export function PasswordPage() {
  const { t } = useI18n()
  const { user, logout } = useAuth()
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setMessage("")
    setError("")
    if (newPassword !== confirmPassword) {
      setError(t("password.mismatch"))
      return
    }
    setPending(true)
    try {
      await api.changePassword({ oldPassword: user?.mustSetPassword ? "" : oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      setConfirmPassword("")
      setMessage(t("password.updated"))
      await logout()
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("password.failed") : t("password.failed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="grid gap-4 rounded-lg border p-6">
      <h1 className="text-xl font-semibold tracking-tight">{t("password.title")}</h1>
      {user?.mustChangePassword ? (
        <p className="text-sm text-destructive">{t("password.mustChange")}</p>
      ) : null}
      {user?.mustSetPassword ? (
        <p className="text-sm text-muted-foreground">{t("password.mustSet")}</p>
      ) : null}
      {user?.mustSetPassword ? null : (
      <div className="grid gap-1.5">
        <label htmlFor="current-password" className="text-sm font-medium">
          {t("password.current")}
        </label>
        <input
          id="current-password"
          type="password"
          autoComplete="current-password"
          value={oldPassword}
          onChange={(e) => setOldPassword(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>
      )}
      <div className="grid gap-1.5">
        <label htmlFor="new-password" className="text-sm font-medium">
          {t("password.next")}
        </label>
        <input
          id="new-password"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>
      <div className="grid gap-1.5">
        <label htmlFor="confirm-password" className="text-sm font-medium">
          {t("password.confirm")}
        </label>
        <input
          id="confirm-password"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>
      {message ? <p className="text-sm text-muted-foreground">{message}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="h-9 rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
      >
        {pending ? t("password.saving") : t("password.save")}
      </button>
    </form>
  )
}
