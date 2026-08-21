import { useEffect } from "react"
import { useBlocker } from "react-router-dom"
import { ConfirmAlert } from "@/components/feedback"
import { useI18n } from "@/providers/i18n"

export function UnsavedGuard({ dirty }: { dirty: boolean }) {
  const { t } = useI18n()
  const blocker = useBlocker(dirty)

  useEffect(() => {
    if (blocker.state === "blocked" && !dirty) {
      blocker.proceed()
    }
  }, [blocker, dirty])

  useEffect(() => {
    if (!dirty) return
    function onLeave(e: BeforeUnloadEvent) {
      e.preventDefault()
      e.returnValue = ""
    }
    window.addEventListener("beforeunload", onLeave)
    return () => window.removeEventListener("beforeunload", onLeave)
  }, [dirty])

  const blocked = blocker.state === "blocked"

  return (
    <ConfirmAlert
      open={blocked}
      onOpenChange={(open) => {
        if (!open && blocker.state === "blocked") blocker.reset()
      }}
      title={t("app.unsavedTitle")}
      description={t("app.unsavedBody")}
      onConfirm={() => {
        if (blocker.state === "blocked") blocker.proceed()
      }}
    />
  )
}
