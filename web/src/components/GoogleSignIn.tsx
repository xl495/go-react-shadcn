import { useEffect, useRef } from "react"
import { loadScript } from "@/lib/load-script"

export function GoogleSignIn({
  clientId,
  onCredential,
  disabled,
}: {
  clientId: string
  onCredential: (idToken: string) => void
  disabled?: boolean
}) {
  const host = useRef<HTMLDivElement>(null)
  const onCredentialRef = useRef(onCredential)
  onCredentialRef.current = onCredential

  useEffect(() => {
    if (!clientId || disabled) return
    if (!host.current) return
    let cancelled = false
    loadScript("https://accounts.google.com/gsi/client")
      .then(() => {
        if (cancelled || !host.current || !window.google?.accounts.id) return
        host.current.innerHTML = ""
        window.google.accounts.id.initialize({
          client_id: clientId,
          ux_mode: "popup",
          callback: (res) => {
            if (res.credential) onCredentialRef.current(res.credential)
          },
        })
        window.google.accounts.id.renderButton(host.current, {
          theme: "outline",
          size: "large",
          width: 384,
          text: "continue_with",
          locale: "zh-CN",
        })
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [clientId, disabled])

  return <div ref={host} className="flex min-h-10 justify-center overflow-hidden rounded-md" />
}
