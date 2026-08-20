import { useEffect, useState, type FormEvent } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from || "/"

  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
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
    loadCaptcha().catch(() => setError("验证码加载失败"))
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const result = await api.login({ username, password, captchaId, captchaCode })
      login(result.token, result.user)
      navigate(from === "/login" ? "/" : from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "登录失败" : "登录失败")
      await loadCaptcha().catch(() => undefined)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="flex h-14 items-center border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
      </header>
      <div className="flex flex-1 items-center justify-center px-4">
        <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">登录</h1>
            <p className="mt-1 text-sm text-muted-foreground">输入账号并完成图形验证码</p>
          </div>
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
          <div className="grid gap-2">
            <label htmlFor="captcha" className="text-sm font-medium">
              验证码
            </label>
            <div className="flex items-center gap-3">
              <input
                id="captcha"
                inputMode="numeric"
                autoComplete="off"
                value={captchaCode}
                onChange={(e) => setCaptchaCode(e.target.value)}
                className="h-10 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
              <button
                type="button"
                onClick={() => loadCaptcha().catch(() => setError("验证码加载失败"))}
                className="relative h-10 w-[120px] overflow-hidden rounded-md border bg-muted"
                aria-label="刷新验证码"
              >
                {image ? (
                  <img src={image} alt="图形验证码" className="h-full w-full object-cover" />
                ) : (
                  <span className="text-xs text-muted-foreground">加载中</span>
                )}
                <RefreshCw className="absolute right-1 bottom-1 size-3 text-foreground/40" />
              </button>
            </div>
          </div>
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <button
            type="submit"
            disabled={loading}
            className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
          >
            {loading ? "登录中…" : "登录"}
          </button>
        </form>
      </div>
    </div>
  )
}
