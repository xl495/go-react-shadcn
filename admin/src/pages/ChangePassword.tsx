import { useState, type FormEvent } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { useAuth } from "@/providers/auth"
import { useI18n } from "@/providers/i18n"
import { useChangePassword } from "@/hooks/queries"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function ChangePasswordPage() {
  const { t } = useI18n()
  const { logout, user } = useAuth()
  const changePassword = useChangePassword()
  const firstSet = Boolean(user?.mustSetPassword)
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error(t("settings.passwordMismatch"))
      return
    }
    try {
      await changePassword.mutateAsync({ oldPassword: firstSet ? "" : oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      setConfirmPassword("")
      toast.success(t("settings.passwordChangedRelogin"))
      await logout()
    } catch {
      // API message is toasted by the HTTP client.
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("settings.password")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t("settings.passwordPageHint")}</p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.password")}</CardTitle>
          <CardDescription>{firstSet ? t("settings.setFirstPassword") : t("settings.passwordHint")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-4">
            {firstSet ? null : (
            <div className="grid gap-1.5">
              <Label htmlFor="old">{t("settings.oldPassword")}</Label>
              <Input
                id="old"
                type="password"
                autoComplete="current-password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
              />
            </div>
            )}
            <div className="grid gap-1.5">
              <Label htmlFor="nw">{t("settings.newPassword")}</Label>
              <Input
                id="nw"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="cf">{t("settings.confirmPassword")}</Label>
              <Input
                id="cf"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={changePassword.isPending}>
                {changePassword.isPending ? t("app.saving") : t("settings.changePassword")}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link to="/settings">{t("settings.backToProfile")}</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
