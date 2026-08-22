import { useEffect, useState, type FormEvent } from "react"
import { Link } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useI18n } from "@/lib/i18n"
import type { AuthSettings, User } from "@/lib/types"
import { GoogleSignIn } from "@/components/GoogleSignIn"

export function ProfilePage() {
  const { user, setUser } = useAuth()
  if (!user) return null
  return <ProfileForm key={user.id} user={user} onSaved={setUser} />
}

function ProfileForm({ user, onSaved }: { user: User; onSaved: (next: User) => void }) {
  const { t, locale } = useI18n()
  const [nickname, setNickname] = useState(user.nickname || "")
  const [email, setEmail] = useState(user.email || "")
  const [phone, setPhone] = useState(user.phone || "")
  const [timezone, setTimezone] = useState(user.timezone || "Asia/Shanghai")
  const [optIn, setOptIn] = useState(user.marketingOptIn ?? true)
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)
  const [settings, setSettings] = useState<AuthSettings | undefined>()
  const [unbindPassword, setUnbindPassword] = useState("")
  const [unbindTotp, setUnbindTotp] = useState("")

  useEffect(() => {
    api.settings().then(setSettings).catch(() => undefined)
  }, [])

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
        gender: user.gender ?? "",
        department: user.department ?? "",
        title: user.title ?? "",
        remark: user.remark ?? "",
        timezone,
        marketingOptIn: optIn,
      })
      onSaved(next)
      setMessage(t("profile.saved"))
    } catch (err) {
      setError(err instanceof ApiError ? err.message || t("profile.failed") : t("profile.failed"))
    } finally {
      setPending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="grid gap-4 rounded-lg border p-6">
      <h1 className="text-xl font-semibold tracking-tight">{t("profile.title")}</h1>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">{t("profile.nickname")}</span>
        <input
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">{t("profile.email")}</span>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
      {user.mustSetPassword ? (
        <p className="text-sm text-muted-foreground">
          {t("password.mustSet")}{" "}
          <Link to="/password" className="text-foreground underline-offset-4 hover:underline">
            {t("nav.password")}
          </Link>
        </p>
      ) : null}
      {user.pendingEmail ? (
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm text-muted-foreground">
            {t("profile.pendingEmail")}: {user.pendingEmail}
          </p>
          <button
            type="button"
            disabled={pending}
            className="h-8 rounded-md border px-2 text-xs hover:bg-muted disabled:opacity-60"
            onClick={() => {
              setError("")
              setMessage("")
              setPending(true)
              void api
                .cancelPendingEmail()
                .then((next) => {
                  onSaved(next)
                  setMessage(t("profile.saved"))
                })
                .catch((err) => {
                  setError(err instanceof ApiError ? err.message || t("profile.failed") : t("profile.failed"))
                })
                .finally(() => setPending(false))
            }}
          >
            {t("profile.cancelPendingEmail")}
          </button>
        </div>
      ) : null}
      <div className="grid gap-2 rounded-md border p-3">
        <p className="text-sm font-medium">{t("profile.google")}</p>
        <p className="text-sm text-muted-foreground">{user.googleBound ? t("profile.googleOn") : t("profile.googleOff")}</p>
        {settings?.googleEnabled ? (
          user.googleBound ? (
            <div className="grid gap-2">
              {user.mustSetPassword ? (
                <p className="text-sm text-muted-foreground">
                  {t("profile.googleNeedPassword")}{" "}
                  <Link to="/password" className="text-foreground underline-offset-4 hover:underline">
                    {t("nav.password")}
                  </Link>
                </p>
              ) : user.totpEnabled ? (
                <label className="grid gap-1.5 text-sm">
                  <span className="text-muted-foreground">{t("profile.googleUnbindTotp")}</span>
                  <input
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    value={unbindTotp}
                    onChange={(e) => setUnbindTotp(e.target.value)}
                    className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  />
                </label>
              ) : (
                <label className="grid gap-1.5 text-sm">
                  <span className="text-muted-foreground">{t("profile.googleUnbindPassword")}</span>
                  <input
                    type="password"
                    autoComplete="current-password"
                    value={unbindPassword}
                    onChange={(e) => setUnbindPassword(e.target.value)}
                    className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  />
                </label>
              )}
              {user.mustSetPassword ? null : (
              <button
                type="button"
                disabled={pending || (user.totpEnabled ? unbindTotp.trim().length < 6 : !unbindPassword)}
                className="h-9 w-fit rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
                onClick={() => {
                  setError("")
                  setMessage("")
                  setPending(true)
                  void api
                    .unbindGoogle(user.totpEnabled ? { totpCode: unbindTotp } : { password: unbindPassword })
                    .then((next) => {
                      onSaved(next)
                      setUnbindPassword("")
                      setUnbindTotp("")
                      setMessage(t("profile.saved"))
                    })
                    .catch((err) => {
                      setError(err instanceof ApiError ? err.message || t("profile.failed") : t("profile.failed"))
                    })
                    .finally(() => setPending(false))
                }}
              >
                {t("profile.googleUnbind")}
              </button>
              )}
            </div>
          ) : (
            <GoogleSignIn
              clientId={settings.googleClientId}
              locale={locale}
              onCredential={(idToken) => {
                setError("")
                setMessage("")
                setPending(true)
                void api
                  .bindGoogle(idToken)
                  .then((next) => {
                    onSaved(next)
                    setMessage(t("profile.saved"))
                  })
                  .catch((err) => {
                    setError(err instanceof ApiError ? err.message || t("profile.failed") : t("profile.failed"))
                  })
                  .finally(() => setPending(false))
              }}
              disabled={pending}
            />
          )
        ) : (
          <p className="text-sm text-muted-foreground">{t("profile.googleDisabled")}</p>
        )}
      </div>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">{t("profile.phone")}</span>
        <input
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>
      <label className="grid gap-1.5 text-sm">
        <span className="text-muted-foreground">{t("profile.timezone")}</span>
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
        {t("profile.marketing")}
      </label>
      {message ? <p className="text-sm text-muted-foreground">{message}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="h-9 rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
      >
        {pending ? t("profile.saving") : t("profile.save")}
      </button>
    </form>
  )
}
