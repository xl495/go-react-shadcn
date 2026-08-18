import { useEffect, useState, type FormEvent } from "react"
import { Navigate, useLocation, useNavigate } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { translateApiError, useI18n } from "@/lib/i18n"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function LoginPage() {
  const { user, login } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from || "/"

  const [username, setUsername] = useState("admin")
  const [password, setPassword] = useState("admin123")
  const [captchaId, setCaptchaId] = useState("")
  const [captchaCode, setCaptchaCode] = useState("")
  const [image, setImage] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function loadCaptcha() {
    const ch = await api.captcha()
    setCaptchaId(ch.captchaId)
    setImage(ch.image)
    setCaptchaCode("")
  }

  useEffect(() => {
    loadCaptcha().catch(() => setError(t("login.captchaLoadFailed")))
  }, [t])

  if (user) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const result = await api.login({ username, password, captchaId, captchaCode })
      login(result.token, result.user)
      navigate(from, { replace: true })
    } catch (err) {
      const message = err instanceof ApiError ? translateApiError(err, t) : t("login.failed")
      setError(message)
      await loadCaptcha().catch(() => undefined)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="flex h-14 items-center justify-between border-b px-6">
        <span className="text-sm font-semibold">Latch</span>
        <LanguageSwitcher />
      </header>
      <div className="flex flex-1 items-center justify-center px-4">
        <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("login.title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("login.subtitle")}</p>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="username">{t("login.username")}</Label>
            <Input
              id="username"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="password">{t("login.password")}</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="captcha">{t("login.captcha")}</Label>
            <div className="flex items-center gap-3">
              <Input
                id="captcha"
                inputMode="numeric"
                autoComplete="off"
                value={captchaCode}
                onChange={(e) => setCaptchaCode(e.target.value)}
                className="flex-1"
              />
              <button
                type="button"
                onClick={() => loadCaptcha().catch(() => setError(t("login.captchaLoadFailed")))}
                className="relative h-9 w-[120px] overflow-hidden rounded-md border bg-muted"
                aria-label={t("login.refreshCaptcha")}
              >
                {image ? (
                  <img src={image} alt={t("login.captchaAlt")} className="h-full w-full object-cover" />
                ) : (
                  <span className="text-xs text-muted-foreground">{t("app.loading")}</span>
                )}
                <RefreshCw className="absolute right-1 bottom-1 size-3 text-foreground/40" />
              </button>
            </div>
          </div>
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <Button type="submit" disabled={loading} className="h-10 w-full">
            {loading ? t("login.submitting") : t("login.submit")}
          </Button>
          <p className="text-xs text-muted-foreground">
            admin / admin123 · operator / operator123 · viewer / viewer123
          </p>
        </form>
      </div>
    </div>
  )
}
