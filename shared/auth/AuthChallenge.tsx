import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react"
import { RefreshCw } from "lucide-react"
import { loadScript } from "./load-script"

export const CAPTCHA_FALLBACK_CODE = 40004

export type ChallengePayload = {
  captchaId?: string
  captchaCode?: string
  captchaToken?: string
  captchaVersion?: string
}

export type AuthChallengeHandle = {
  collect: () => Promise<ChallengePayload>
  showV2: () => void
}

export type AuthChallengeSettings = {
  captchaProvider?: string
  recaptchaSiteKeyV3?: string
  recaptchaSiteKeyV2?: string
  turnstileSiteKey?: string
}

const COPY = {
  admin: {
    captcha: "login.captcha",
    refresh: "login.refreshCaptcha",
    alt: "login.captchaAlt",
    loading: "app.loading",
    failed: "login.captchaLoadFailed",
    widget: "login.captcha",
    fallback: "login.captchaFallback",
    recaptcha: "login.recaptchaHint",
  },
  web: {
    captcha: "captcha.label",
    refresh: "captcha.refresh",
    alt: "captcha.imageAlt",
    loading: "captcha.loading",
    failed: "captcha.failed",
    widget: "captcha.verify",
    fallback: "captcha.complete",
    recaptcha: "captcha.recaptcha",
  },
} as const

export const AuthChallenge = forwardRef<
  AuthChallengeHandle,
  {
    settings?: AuthChallengeSettings
    action: string
    t: (key: string) => string
    variant: "admin" | "web"
    loadCaptcha: () => Promise<{ captchaId: string; image: string }>
  }
>(function AuthChallenge({ settings, action, t, variant, loadCaptcha }, ref) {
  const copy = COPY[variant]
  const provider = settings?.captchaProvider
  const v3Key = settings?.recaptchaSiteKeyV3 ?? ""
  const v2Key = settings?.recaptchaSiteKeyV2 ?? ""
  const [forceV2, setForceV2] = useState(false)
  const useV2 = provider === "recaptcha" && (forceV2 || !v3Key) && !!v2Key

  const [captchaId, setCaptchaId] = useState("")
  const [captchaCode, setCaptchaCode] = useState("")
  const [image, setImage] = useState("")
  const [imageError, setImageError] = useState(false)
  const widgetToken = useRef("")
  const widgetHost = useRef<HTMLDivElement>(null)
  const widgetId = useRef<string | number>("")

  async function refreshImage() {
    setCaptchaCode("")
    setImageError(false)
    const ch = await loadCaptcha()
    setCaptchaId(ch.captchaId)
    setImage(ch.image)
  }

  useEffect(() => {
    if (provider !== "image") return
    refreshImage().catch(() => setImageError(true))
  }, [provider])

  useEffect(() => {
    widgetToken.current = ""
    let cancelled = false

    async function mount() {
      if (provider === "recaptcha" && v3Key && !useV2) {
        await loadScript(`https://www.google.com/recaptcha/api.js?render=${encodeURIComponent(v3Key)}`)
        return
      }
      if (!widgetHost.current) return
      if (provider === "turnstile" && settings?.turnstileSiteKey) {
        await loadScript("https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit")
        if (cancelled || !widgetHost.current || !window.turnstile) return
        widgetHost.current.innerHTML = ""
        widgetId.current = window.turnstile.render(widgetHost.current, {
          sitekey: settings.turnstileSiteKey,
          callback: (token) => {
            widgetToken.current = token
          },
          "expired-callback": () => {
            widgetToken.current = ""
          },
        })
        return
      }
      if (provider === "recaptcha" && useV2 && v2Key) {
        await loadScript("https://www.google.com/recaptcha/api.js?render=explicit")
        if (cancelled || !widgetHost.current || !window.grecaptcha) return
        await new Promise<void>((resolve) => window.grecaptcha?.ready(() => resolve()))
        if (cancelled || !widgetHost.current || !window.grecaptcha) return
        widgetHost.current.innerHTML = ""
        widgetId.current = window.grecaptcha.render(widgetHost.current, {
          sitekey: v2Key,
          callback: (token) => {
            widgetToken.current = token
          },
          "expired-callback": () => {
            widgetToken.current = ""
          },
        })
      }
    }

    mount().catch(() => undefined)
    return () => {
      cancelled = true
      if (provider === "turnstile" && widgetId.current && window.turnstile) {
        window.turnstile.remove(String(widgetId.current))
      }
    }
  }, [provider, useV2, v2Key, v3Key, settings?.turnstileSiteKey])

  useImperativeHandle(
    ref,
    () => ({
      showV2: () => setForceV2(true),
      collect: async () => {
        if (!provider || provider === "none") return {}
        if (provider === "image") return { captchaId, captchaCode }
        if (provider === "turnstile") return { captchaToken: widgetToken.current }
        if (useV2) return { captchaToken: widgetToken.current, captchaVersion: "v2" }
        if (!v3Key || !window.grecaptcha) return { captchaToken: "", captchaVersion: "v3" }
        const token = await new Promise<string>((resolve, reject) => {
          window.grecaptcha?.ready(() => {
            window.grecaptcha
              ?.execute(v3Key, { action })
              .then(resolve)
              .catch(() => reject(new Error("recaptcha")))
          })
        })
        return { captchaToken: token, captchaVersion: "v3" }
      },
    }),
    [action, captchaCode, captchaId, provider, useV2, v3Key],
  )

  const fieldClass =
    variant === "admin"
      ? "h-9 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      : "h-10 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
  const imageBtnClass =
    variant === "admin"
      ? "relative h-9 w-[120px] overflow-hidden rounded-md border p-0"
      : "relative h-10 w-[120px] overflow-hidden rounded-md border bg-muted"

  if (!settings) {
    return <div className="h-10 rounded-md bg-muted/50" aria-hidden />
  }

  if (provider === "none") return null

  if (provider === "image") {
    return (
      <div className="grid gap-2">
        <label htmlFor="captcha" className="text-sm font-medium">
          {t(copy.captcha)}
        </label>
        <div className="flex items-center gap-3">
          <input
            id="captcha"
            inputMode="numeric"
            autoComplete="off"
            value={captchaCode}
            onChange={(e) => setCaptchaCode(e.target.value)}
            className={fieldClass}
          />
          <button
            type="button"
            onClick={() => void refreshImage().catch(() => setImageError(true))}
            className={imageBtnClass}
            aria-label={t(copy.refresh)}
          >
            {image ? (
              <img src={image} alt={t(copy.alt)} className="h-full w-full object-cover" />
            ) : (
              <span className="text-xs text-muted-foreground">{t(copy.loading)}</span>
            )}
            <RefreshCw className="absolute right-1 bottom-1 size-3 text-foreground/40" />
          </button>
        </div>
        {imageError ? <p className="text-sm text-destructive">{t(copy.failed)}</p> : null}
      </div>
    )
  }

  if (provider === "turnstile") {
    return (
      <div className="grid gap-2">
        <p className="text-sm font-medium">{t(copy.widget)}</p>
        <div ref={widgetHost} className="min-h-[65px]" />
      </div>
    )
  }

  if (provider === "recaptcha" && useV2) {
    return (
      <div className="grid gap-2">
        <p className="text-sm font-medium">{t(copy.fallback)}</p>
        <div ref={widgetHost} className="min-h-[78px]" />
      </div>
    )
  }

  if (provider === "recaptcha") {
    return <p className="text-xs text-muted-foreground">{t(copy.recaptcha)}</p>
  }

  return null
})
