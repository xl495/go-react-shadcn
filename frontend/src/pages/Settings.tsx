import { useEffect, useState, type FormEvent } from "react"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { DICT, useDict } from "@/lib/dict"
import { formatDateTime } from "@/lib/format"
import { translateApiError, useI18n } from "@/lib/i18n"
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
  const [profileMsg, setProfileMsg] = useState("")
  const [profileErr, setProfileErr] = useState("")
  const [pwdMsg, setPwdMsg] = useState("")
  const [pwdErr, setPwdErr] = useState("")
  const [saving, setSaving] = useState(false)
  const [changing, setChanging] = useState(false)

  useEffect(() => {
    setProfile({
      nickname: user?.nickname ?? "",
      email: user?.email ?? "",
      phone: user?.phone ?? "",
      gender: user?.gender ?? "",
      department: user?.department ?? "",
      title: user?.title ?? "",
      remark: user?.remark ?? "",
    })
  }, [user])

  async function onSaveProfile(e: FormEvent) {
    e.preventDefault()
    setProfileErr("")
    setProfileMsg("")
    setSaving(true)
    try {
      const next = await api.updateProfile(profile)
      updateUser(next)
      setProfileMsg(t("settings.saved"))
    } catch (err) {
      setProfileErr(translateApiError(err, t))
    } finally {
      setSaving(false)
    }
  }

  async function onChangePassword(e: FormEvent) {
    e.preventDefault()
    setPwdErr("")
    setPwdMsg("")
    setChanging(true)
    try {
      await api.changePassword({ oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      setPwdMsg(t("settings.passwordChanged"))
    } catch (err) {
      setPwdErr(translateApiError(err, t))
    } finally {
      setChanging(false)
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
            {profileErr ? <p className="text-sm text-destructive sm:col-span-2">{profileErr}</p> : null}
            {profileMsg ? <p className="text-sm text-foreground sm:col-span-2">{profileMsg}</p> : null}
            <div className="sm:col-span-2">
              <Button type="submit" disabled={saving}>
                {saving ? t("app.saving") : t("settings.saveProfile")}
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
            {pwdErr ? <p className="text-sm text-destructive">{pwdErr}</p> : null}
            {pwdMsg ? <p className="text-sm">{pwdMsg}</p> : null}
            <Button type="submit" disabled={changing}>
              {changing ? t("app.saving") : t("settings.changePassword")}
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
