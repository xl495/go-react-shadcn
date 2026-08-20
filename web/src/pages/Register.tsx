import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import type { AuthSettings } from "@/lib/types"
import { GoogleSignIn } from "@/components/GoogleSignIn"

export function RegisterPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [settings, setSettings] = useState<AuthSettings | undefined>()
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api.settings().then(setSettings).catch(() => undefined)
  }, [])

  async function onGoogle(idToken: string) {
    setError("")
    setLoading(true)
    try {
      const result = await api.google({ idToken, client: "web" })
      login(result.token, result.user)
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "注册失败" : "注册失败")
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
        <div className="w-full max-w-sm space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">注册</h1>
            <p className="mt-1 text-sm text-muted-foreground">使用 Google 账号创建用户端账号</p>
          </div>
          {settings && !settings.googleEnabled ? (
            <p className="text-sm text-muted-foreground">尚未开启 Google 登录</p>
          ) : null}
          {settings?.googleEnabled && !settings.googleRegisterEnabled ? (
            <p className="text-sm text-muted-foreground">当前未开放 Google 注册</p>
          ) : null}
          {settings?.googleEnabled && settings.googleRegisterEnabled ? (
            <GoogleSignIn clientId={settings.googleClientId} onCredential={onGoogle} disabled={loading} />
          ) : null}
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <Link to="/login" className="text-sm text-primary underline-offset-4 hover:underline">
            返回登录
          </Link>
        </div>
      </div>
    </div>
  )
}
