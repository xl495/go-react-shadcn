import { useState, type FormEvent } from "react"
import { toast } from "sonner"
import { api } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { DICT, useDict } from "@/hooks/dict"
import { formatDateTime } from "@/utils/format"
import { translateApiError, useI18n } from "@/providers/i18n"
import { useChangePassword, useUpdateProfile } from "@/hooks/queries"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { AvatarField } from "@/components/ui/avatar-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DictSelect } from "@/components/ui/dict-select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function SettingsPage() {
  const { user, updateUser } = useAuth()
  const { t } = useI18n()
  const genderDict = useDict(DICT.gender)
  const deptDict = useDict(DICT.department)
  const updateProfile = useUpdateProfile()
  const changePassword = useChangePassword()
  const [profile, setProfile] = useState({
    nickname: user?.nickname ?? "",
    email: user?.email ?? "",
    phone: user?.phone ?? "",
    gender: user?.gender ?? "",
    department: user?.department ?? "",
    title: user?.title ?? "",
    remark: user?.remark ?? "",
  })
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")

  async function onSaveProfile(e: FormEvent) {
    e.preventDefault()
    try {
      const next = await updateProfile.mutateAsync(profile)
      updateUser(next)
      toast.success(t("settings.saved"))
    } catch (err) {
      toast.error(translateApiError(err, t))
    }
  }

  async function onChangePassword(e: FormEvent) {
    e.preventDefault()
    try {
      await changePassword.mutateAsync({ oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      toast.success(t("settings.passwordChanged"))
    } catch (err) {
      toast.error(translateApiError(err, t))
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{t("settings.title")}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t("settings.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.profile")}</CardTitle>
          <CardDescription>
            {user?.username} · {t("users.lastLogin")} {formatDateTime(user?.lastLoginAt)}
            {user?.lastLoginIp ? ` · ${user.lastLoginIp}` : ""}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSaveProfile} className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <AvatarField
                name={user?.nickname || user?.username}
                src={user?.avatar}
                onFile={async (file) => {
                  const next = await api.uploadOwnAvatar(file)
                  updateUser(next)
                  toast.success(t("app.saved"))
                }}
              />
            </div>
            {(
              [
                ["nickname", "users.nickname"],
                ["phone", "users.phone"],
                ["email", "users.email"],
                ["title", "users.jobTitle"],
                ["remark", "users.remark"],
              ] as const
            ).map(([key, label]) => (
              <div key={key} className="grid gap-1.5">
                <Label htmlFor={key}>{t(label)}</Label>
                <Input
                  id={key}
                  value={profile[key]}
                  onChange={(e) => setProfile((p) => ({ ...p, [key]: e.target.value }))}
                />
              </div>
            ))}
            <DictSelect
              id="gender"
              label={t("users.gender")}
              value={profile.gender}
              items={genderDict.items}
              allowEmpty
              emptyLabel={t("users.genderUnset")}
              onChange={(value) => setProfile((p) => ({ ...p, gender: value }))}
            />
            <DictSelect
              id="department"
              label={t("users.department")}
              value={profile.department}
              items={deptDict.items}
              allowEmpty
              emptyLabel={t("app.optional")}
              onChange={(value) => setProfile((p) => ({ ...p, department: value }))}
            />
            <div className="sm:col-span-2">
              <Button type="submit" disabled={updateProfile.isPending}>
                {updateProfile.isPending ? t("app.saving") : t("settings.saveProfile")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.password")}</CardTitle>
          <CardDescription>{t("settings.passwordHint")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onChangePassword} className="grid max-w-md gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="old">{t("settings.oldPassword")}</Label>
              <Input
                id="old"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="nw">{t("settings.newPassword")}</Label>
              <Input
                id="nw"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={changePassword.isPending}>
              {changePassword.isPending ? t("app.saving") : t("settings.changePassword")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("app.language")}</CardTitle>
          <CardDescription>{t("settings.languageHint")}</CardDescription>
        </CardHeader>
        <CardContent>
          <LanguageSwitcher />
        </CardContent>
      </Card>
    </div>
  )
}
