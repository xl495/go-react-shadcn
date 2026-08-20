import { useRef, useState } from "react"
import { Avatar } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/lib/i18n"

export function AvatarField({
  name,
  src,
  onFile,
}: {
  name?: string
  src?: string
  onFile: (file: File) => Promise<void>
}) {
  const { t } = useI18n()
  const inputRef = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState("")

  return (
    <div className="flex items-center gap-4">
      <Avatar name={name} src={src} className="size-16 text-lg" />
      <div className="space-y-1.5">
        <input
          ref={inputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0]
            e.target.value = ""
            if (!file) return
            setErr("")
            setBusy(true)
            onFile(file)
              .catch((error: Error) => setErr(error.message))
              .finally(() => setBusy(false))
          }}
        />
        <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => inputRef.current?.click()}>
          {busy ? t("app.saving") : t("users.changeAvatar")}
        </Button>
        <p className="text-xs text-muted-foreground">{t("users.avatarHint")}</p>
        {err ? <p className="text-xs text-destructive">{err}</p> : null}
      </div>
    </div>
  )
}
