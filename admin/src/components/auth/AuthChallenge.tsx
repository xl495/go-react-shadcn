import { forwardRef } from "react"
import {
  AuthChallenge as SharedAuthChallenge,
  CAPTCHA_FALLBACK_CODE,
  type AuthChallengeHandle,
  type AuthChallengeSettings,
  type ChallengePayload,
} from "@latch/auth/AuthChallenge"
import { api } from "@/api/client"
import { useI18n } from "@/providers/i18n"

export { CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle, type ChallengePayload }

export const AuthChallenge = forwardRef<
  AuthChallengeHandle,
  { settings?: AuthChallengeSettings; action: string; t?: (key: string) => string }
>(function AuthChallenge({ settings, action, t: tProp }, ref) {
  const { t } = useI18n()
  return (
    <SharedAuthChallenge
      ref={ref}
      settings={settings}
      action={action}
      t={tProp ?? t}
      variant="admin"
      loadCaptcha={api.captcha}
    />
  )
})
