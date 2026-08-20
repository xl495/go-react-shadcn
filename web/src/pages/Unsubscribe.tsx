import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { api, ApiError } from "@/lib/api"

export function UnsubscribePage() {
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""
  const [pending, setPending] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState("")

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setPending(true)
    try {
      await api.unsubscribe(token)
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message || "退订失败" : "退订失败")
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex h-full flex-col overflow-y-auto bg-background text-foreground">
      <header className="flex h-14 items-center justify-between border-b px-6">
        <span className="text-sm font-semibold tracking-tight">Latch</span>
        <Link to="/login" className="text-sm text-muted-foreground hover:text-foreground">
          登录
        </Link>
      </header>
      <main className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center px-4 py-10">
        {done ? (
          <section className="space-y-3">
            <h1 className="text-2xl font-semibold tracking-tight">退订营销邮件</h1>
            <p className="text-sm text-muted-foreground">已退订。事务与运营通知不受影响。</p>
          </section>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">退订营销邮件</h1>
              <p className="mt-1 text-sm text-muted-foreground">确认后将不再向此账号发送营销邮件。</p>
            </div>
            {!token ? <p className="text-sm text-destructive">退订链接无效。</p> : null}
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <button
              type="submit"
              disabled={pending || !token}
              className="h-10 w-full rounded-md border px-3 text-sm hover:bg-muted disabled:opacity-60"
            >
              {pending ? "提交中…" : "确认退订"}
            </button>
          </form>
        )}
      </main>
    </div>
  )
}
