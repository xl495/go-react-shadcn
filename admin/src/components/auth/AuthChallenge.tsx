import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react"
import { RefreshCw } from "lucide-react"
import { api } from "@/api/client"
import { loadScript } from "@/lib/load-script"
import type { AuthSettings } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

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

export const AuthChallenge = forwardRef<
  AuthChallengeHandle,
  { settings?: AuthSettings; action: string; t: (key: string) => string }
>(function AuthChallenge({ settings, action, t }, ref) {
  const provider = settings?.captchaProvider ?? "image"
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

  async function loadImage() {
    setCaptchaCode("")
    setImageError(false)
    const ch = await api.captcha()
    setCaptchaId(ch.captchaId)
    setImage(ch.image)
  }

  useEffect(() => {
    if (provider !== "image") return
    loadImage().catch(() => setImageError(true))
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
        if (provider === "none") return {}
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

  if (provider === "none") return null

  if (provider === "image") {
    return (
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
          <Button
            type="button"
            variant="outline"
            onClick={() => void loadImage().catch(() => setImageError(true))}
            className="relative h-9 w-[120px] overflow-hidden p-0"
            aria-label={t("login.refreshCaptcha")}
          >
            {image ? (
              <img src={image} alt={t("login.captchaAlt")} className="h-full w-full object-cover" />
            ) : (
              <span className="text-xs text-muted-foreground">{t("app.loading")}</span>
            )}
            <RefreshCw className="absolute right-1 bottom-1 size-3 text-foreground/40" />
          </Button>
        </div>
        {imageError ? <p className="text-sm text-destructive">{t("login.captchaLoadFailed")}</p> : null}
      </div>
    )
  }

  if (provider === "turnstile") {
    return (
      <div className="grid gap-2">
        <p className="text-sm font-medium">{t("login.captcha")}</p>
        <div ref={widgetHost} className="min-h-[65px]" />
      </div>
    )
  }

  if (provider === "recaptcha" && useV2) {
    return (
      <div className="grid gap-2">
        <p className="text-sm font-medium">{t("login.captchaFallback")}</p>
        <div ref={widgetHost} className="min-h-[78px]" />
      </div>
    )
  }

  if (provider === "recaptcha") {
    return <p className="text-xs text-muted-foreground">{t("login.recaptchaHint")}</p>
  }

  return null
})
