import { useState, type FormEvent } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { api } from "@/api/client"
import { useAuth } from "@/providers/auth"
import { DICT, useDict } from "@/hooks/dict"
import { formatDateTime } from "@/utils/format"
import { useI18n } from "@/providers/i18n"
import { useUpdateProfile } from "@/hooks/queries"
import { UnsavedGuard } from "@/hooks/unsaved"
import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher"
import { useTheme, type Theme } from "@/providers/theme"
import { AvatarField } from "@/components/ui/avatar-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DictSelect } from "@/components/ui/dict-select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { TimezoneSelect } from "@/components/ui/timezone-select"

export function SettingsPage() {
  const { user, updateUser } = useAuth()
  const { t } = useI18n()
  const { theme, setTheme } = useTheme()
  const genderDict = useDict(DICT.gender)
  const deptDict = useDict(DICT.department)
  const updateProfile = useUpdateProfile()
  const [profile, setProfile] = useState({
    nickname: user?.nickname ?? "",
    email: user?.email ?? "",
    phone: user?.phone ?? "",
    gender: user?.gender ?? "",
    department: user?.department ?? "",
    title: user?.title ?? "",
    remark: user?.remark ?? "",
    timezone: user?.timezone || "Asia/Shanghai",
    marketingOptIn: user?.marketingOptIn ?? true,
  })
  const dirty =
    profile.nickname !== (user?.nickname ?? "") ||
    profile.email !== (user?.email ?? "") ||
    profile.phone !== (user?.phone ?? "") ||
    profile.gender !== (user?.gender ?? "") ||
    profile.department !== (user?.department ?? "") ||
    profile.title !== (user?.title ?? "") ||
    profile.remark !== (user?.remark ?? "") ||
    profile.timezone !== (user?.timezone || "Asia/Shanghai") ||
    profile.marketingOptIn !== (user?.marketingOptIn ?? true)

  async function onSaveProfile(e: FormEvent) {
    e.preventDefault()
    try {
      const next = await updateProfile.mutateAsync(profile)
      updateUser(next)
      toast.success(t("settings.saved"))
    } catch {
      // API message is toasted by the HTTP client.
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <UnsavedGuard dirty={dirty} />
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{t("settings.title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("settings.subtitle")}</p>
        </div>
        <Button asChild variant="outline">
          <Link to="/settings/password">{t("settings.password")}</Link>
        </Button>
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
            <div className="grid gap-1.5">
              <Label htmlFor="tz">{t("settings.timezone")}</Label>
              <TimezoneSelect
                id="tz"
                value={profile.timezone}
                onChange={(timezone) => setProfile((p) => ({ ...p, timezone }))}
              />
            </div>
            <div className="flex items-center gap-2 sm:col-span-2">
              <Switch
                id="mkt"
                checked={profile.marketingOptIn}
                onCheckedChange={(checked) => setProfile((p) => ({ ...p, marketingOptIn: checked }))}
              />
              <Label htmlFor="mkt" className="font-normal">
                {t("settings.marketingOptIn")}
              </Label>
            </div>
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
          <CardTitle>{t("settings.appearance")}</CardTitle>
          <CardDescription>{t("settings.appearanceHint")}</CardDescription>
        </CardHeader>
        <CardContent>
          <DictSelect
            id="theme"
            label={t("nav.theme")}
            value={theme}
            items={[
              { value: "system", label: t("nav.themeMode.system") },
              { value: "light", label: t("nav.themeMode.light") },
              { value: "dark", label: t("nav.themeMode.dark") },
            ]}
            onChange={(value) => setTheme(value as Theme)}
          />
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
