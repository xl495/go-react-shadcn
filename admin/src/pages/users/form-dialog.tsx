import type { Dispatch, SetStateAction } from "react"
import { toast } from "sonner"
import { api } from "@/api/client"
import { roleLabel, useI18n } from "@/providers/i18n"
import { AvatarField } from "@/components/ui/avatar-field"
import { DictSelect } from "@/components/ui/dict-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { TimezoneSelect } from "@/components/ui/timezone-select"
import type { DictLookup, Role, User } from "@/types"

export const emptyUserForm = {
  username: "",
  password: "",
  nickname: "",
  email: "",
  phone: "",
  gender: "",
  department: "",
  title: "",
  remark: "",
  timezone: "Asia/Shanghai",
  marketingOptIn: false,
  status: "active",
}

export type UserFormValues = typeof emptyUserForm

export function UserFormDialog({
  open,
  onOpenChange,
  kind,
  editing,
  form,
  setForm,
  roles,
  roleIds,
  onToggleRole,
  onSubmit,
  onAvatar,
  canAssignRoles,
  genderItems,
  statusItems,
  deptItems,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  kind: "admin" | "web"
  editing: User | null
  form: UserFormValues
  setForm: Dispatch<SetStateAction<UserFormValues>>
  roles: Role[]
  roleIds: number[]
  onToggleRole: (id: number) => void
  onSubmit: () => void
  onAvatar?: (user: User) => void
  canAssignRoles: boolean
  genderItems: DictLookup["items"]
  statusItems: DictLookup["items"]
  deptItems: DictLookup["items"]
}) {
  const { t } = useI18n()
  const isWeb = kind === "web"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? t("users.edit") : isWeb ? t("webUsers.create") : t("users.create")}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-3 sm:grid-cols-2">
          {editing ? (
            <div className="sm:col-span-2">
              <AvatarField
                name={editing.nickname || editing.username}
                src={editing.avatar}
                onFile={async (file) => {
                  const next = await api.uploadUserAvatar(editing.id, file, kind)
                  onAvatar?.(next)
                  toast.success(t("app.saved"))
                }}
              />
            </div>
          ) : null}
          <div className="grid gap-1.5">
            <Label htmlFor="nu">{t("login.username")}</Label>
            <Input
              id="nu"
              value={form.username}
              disabled={!!editing}
              onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="np">{t("login.password")}</Label>
            <Input
              id="np"
              type="password"
              placeholder={editing ? t("users.passwordKeep") : ""}
              value={form.password}
              onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
            />
          </div>
          {(isWeb
            ? ([
                ["nickname", "users.nickname"],
                ["phone", "users.phone"],
                ["email", "users.email"],
                ["remark", "users.remark"],
              ] as const)
            : ([
                ["nickname", "users.nickname"],
                ["phone", "users.phone"],
                ["email", "users.email"],
                ["title", "users.jobTitle"],
                ["remark", "users.remark"],
              ] as const)
          ).map(([key, label]) => (
            <div key={key} className="grid gap-1.5">
              <Label htmlFor={key}>{t(label)}</Label>
              <Input
                id={key}
                value={form[key]}
                onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
              />
            </div>
          ))}
          <DictSelect
            id="gd"
            label={t("users.gender")}
            value={form.gender}
            items={genderItems}
            allowEmpty
            emptyLabel={t("users.genderUnset")}
            onChange={(value) => setForm((f) => ({ ...f, gender: value }))}
          />
          {!isWeb ? (
            <DictSelect
              id="dept"
              label={t("users.department")}
              value={form.department}
              items={deptItems}
              allowEmpty
              emptyLabel={t("app.optional")}
              onChange={(value) => setForm((f) => ({ ...f, department: value }))}
            />
          ) : null}
          <DictSelect
            id="st"
            label={t("app.status")}
            value={form.status}
            items={statusItems}
            onChange={(value) => setForm((f) => ({ ...f, status: value }))}
          />
          <div className="grid gap-1.5">
            <Label htmlFor="utz">{t("users.timezone")}</Label>
            <TimezoneSelect
              id="utz"
              value={form.timezone}
              onChange={(timezone) => setForm((f) => ({ ...f, timezone }))}
            />
          </div>
          <div className="flex items-center gap-2 text-sm sm:col-span-2">
            <Switch
              id="umkt"
              checked={form.marketingOptIn}
              onCheckedChange={(checked) => setForm((f) => ({ ...f, marketingOptIn: checked }))}
            />
            <Label htmlFor="umkt" className="font-normal">
              {t("users.marketingOptIn")}
            </Label>
          </div>
          {canAssignRoles && (
            <div className="grid gap-2 sm:col-span-2">
              <Label>{t("users.roles")}</Label>
              {roles.map((r) => {
                const id = `ur-${r.id}`
                return (
                  <div key={r.id} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      id={id}
                      checked={roleIds.includes(r.id)}
                      onCheckedChange={() => onToggleRole(r.id)}
                    />
                    <label htmlFor={id} className="cursor-pointer">
                      {roleLabel(r.code, r.name, t)}
                      <span className="ml-1 text-muted-foreground">({r.code})</span>
                    </label>
                  </div>
                )
              })}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button onClick={() => void onSubmit()}>{editing ? t("app.save") : t("app.create")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
