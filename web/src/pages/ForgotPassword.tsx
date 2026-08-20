import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import type { AuthSettings } from "@/lib/types"
import { AuthChallenge, CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle } from "@/components/AuthChallenge"

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("")
  const [error, setError] = useState("")
  const [done, setDone] = useState(false)
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
      await api.forgotPassword({ email, ...challenge })
      setDone(true)
    } catch (err) {
      if (err instanceof ApiError && err.code === CAPTCHA_FALLBACK_CODE) {
        challengeRef.current?.showV2()
      }
      setError(err instanceof ApiError ? err.message || "发送失败" : "发送失败")
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
            <AuthChallenge ref={challengeRef} settings={settings} action="forgot" />
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

function GuestFrame({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-full flex-col overflow-y-auto bg-background text-foreground">
      <header className="flex h-14 items-center border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
      </header>
      <div className="flex flex-1 items-center justify-center px-4">{children}</div>
    </div>
  )
}
