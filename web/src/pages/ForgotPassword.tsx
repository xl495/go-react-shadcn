import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { api, ApiError } from "@/lib/api"

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("")
  const [captchaId, setCaptchaId] = useState("")
  const [captchaCode, setCaptchaCode] = useState("")
  const [image, setImage] = useState("")
  const [error, setError] = useState("")
  const [done, setDone] = useState(false)
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
      await api.forgotPassword({ email, captchaId, captchaCode })
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "发送失败" : "发送失败")
      await loadCaptcha().catch(() => undefined)
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestFrame>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">找回密码</h1>
          <p className="mt-1 text-sm text-muted-foreground">输入账号绑定的邮箱，我们会发送重置链接</p>
        </div>
        {done ? (
          <p className="text-sm text-muted-foreground">如果该邮箱已注册，你将收到重置邮件。</p>
        ) : (
          <>
            <div className="grid gap-2">
              <label htmlFor="email" className="text-sm font-medium">
                邮箱
              </label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
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
              {loading ? "发送中…" : "发送重置邮件"}
            </button>
          </>
        )}
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          返回登录
        </Link>
      </form>
    </GuestFrame>
  )
}

export function ResetPasswordPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    if (!token) {
      setError("重置链接无效或已过期")
      return
    }
    setLoading(true)
    try {
      await api.resetPassword({ token, newPassword: password })
      navigate("/login", { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "重置失败" : "重置失败")
    } finally {
      setLoading(false)
    }
  }

  return (
    <GuestFrame>
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-5">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">设置新密码</h1>
          <p className="mt-1 text-sm text-muted-foreground">请设置至少 8 位的新密码</p>
        </div>
        {!token ? <p className="text-sm text-destructive">重置链接无效或已过期</p> : null}
        <div className="grid gap-2">
          <label htmlFor="np" className="text-sm font-medium">
            新密码
          </label>
          <input
            id="np"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <button
          type="submit"
          disabled={loading || !token}
          className="h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground disabled:opacity-60"
        >
          {loading ? "提交中…" : "重置密码"}
        </button>
        <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          返回登录
        </Link>
      </form>
    </GuestFrame>
  )
}

function GuestFrame({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="flex h-14 items-center border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
      </header>
      <div className="flex flex-1 items-center justify-center px-4">{children}</div>
    </div>
  )
}
