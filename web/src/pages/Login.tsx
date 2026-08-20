import { useEffect, useRef, useState, type FormEvent } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import type { AuthSettings } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"
import { GoogleSignIn } from "@/components/GoogleSignIn"

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from || "/"

  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [settings, setSettings] = useState<AuthSettings | undefined>()
  const challengeRef = useRef<AuthChallengeHandle>(null)

  useEffect(() => {
    api.settings().then(setSettings).catch(() => undefined)
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const challenge = (await challengeRef.current?.collect()) ?? {}
      const result = await api.login({ username, password, ...challenge })
      login(result.token, result.user)
      navigate(from === "/login" || from === "/register" ? "/" : from, { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      setError(err instanceof ApiError ? err.message || "登录失败" : "登录失败")
    } finally {
      setLoading(false)
    }
  }

  async function onGoogle(idToken: string) {
    setError("")
    setLoading(true)
    try {
      const result = await api.google({ idToken, client: "web" })
      login(result.token, result.user)
      navigate(from === "/login" || from === "/register" ? "/" : from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "登录失败" : "登录失败")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex h-full flex-col overflow-y-auto bg-background text-foreground">
      <header className="flex h-14 items-center border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
      </header>
      <div className="flex flex-1 items-center justify-center px-4">
        <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">登录</h1>
            <p className="mt-1 text-sm text-muted-foreground">输入账号并完成人机验证</p>
          </div>
          {settings?.googleEnabled ? (
            <div className="space-y-3">
              <GoogleSignIn clientId={settings.googleClientId} onCredential={onGoogle} disabled={loading} />
              {settings.googleRegisterEnabled ? (
                <p className="text-center text-xs text-muted-foreground">未注册的 Google 账号将自动创建</p>
              ) : null}
              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span className="h-px flex-1 bg-border" />
                或
                <span className="h-px flex-1 bg-border" />
              </div>
            </div>
          ) : null}
          <div className="grid gap-2">
            <label htmlFor="username" className="text-sm font-medium">
              用户名
            </label>
            <input
              id="username"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="h-10 rounded-md border bg-background px-3 text-sm outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring"
            />
          </div>
          <div className="grid gap-2">
            <label htmlFor="password" className="text-sm font-medium">
              密码
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
            <a href="/forgot-password" className="text-xs text-primary underline-offset-4 hover:underline">
              忘记密码
            </a>
          </div>
          <AuthChallenge ref={challengeRef} settings={settings} action="login" />
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <button
            type="submit"
            disabled={loading}
            className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
          >
            {loading ? "登录中…" : "登录"}
          </button>
          {settings?.googleRegisterEnabled ? (
            <p className="text-center text-sm text-muted-foreground">
              还没有账号？{" "}
              <a href="/register" className="text-foreground underline-offset-4 hover:underline">
                注册
              </a>
            </p>
          ) : null}
        </form>
      </div>
    </div>
  )
}
