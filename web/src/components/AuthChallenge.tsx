import { forwardRef } from "react"
import {
  AuthChallenge as SharedAuthChallenge,
  CAPTCHA_FALLBACK_CODE,
  type AuthChallengeHandle,
  type AuthChallengeSettings,
  type ChallengePayload,
} from "@latch/auth/AuthChallenge"
import { api } from "@/lib/api"
import { useI18n } from "@/lib/i18n"

export { CAPTCHA_FALLBACK_CODE, type AuthChallengeHandle, type ChallengePayload }

export const AuthChallenge = forwardRef<
  AuthChallengeHandle,
  { settings?: AuthChallengeSettings; action: string }
>(function AuthChallenge({ settings, action }, ref) {
  const { t } = useI18n()
  return (
    <SharedAuthChallenge
      ref={ref}
      settings={settings}
      action={action}
      t={t}
      variant="web"
      loadCaptcha={api.captcha}
    />
  )
})
